package combat_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func msInitiative(score, mod int) func(*content.MonsterFields) {
	return func(f *content.MonsterFields) { f.Initiative = luMust(content.NewStatBlockInitiative(mod, score)) }
}

// Requirement: doc/specs/monster-combatant-setup.md FR-025 to FR-027, table 4.5a - the mode defaults to rolled; the rolled plan carries the printed modifier and the fixed plan the printed score as the count, never 10 plus the modifier (SRD 5.2.1, "Initiative", p. 254; p. 184).
// Path: per case.
func TestMonsterSetupInitiativePlan(t *testing.T) {
	t.Parallel()
	rolled := func(m int) combat.MonsterInitiativePlan {
		return combat.RolledMonsterInitiative{Modifier: modifier.New(m)}
	}
	fixed := func(c int) combat.MonsterInitiativePlan {
		return combat.FixedMonsterInitiative{Count: combat.InitiativeCount(c)}
	}
	cases := []struct {
		name, requirement, path string
		edit                    func(*content.MonsterFields)
		options                 []combat.SetupOption
		want                    combat.MonsterInitiativePlan
	}{
		{"typical, rolled", "FR-026", "happy", nil, []combat.SetupOption{combat.WithInitiativeMode(combat.MonsterInitiativeRolled)}, rolled(7)},
		{"typical, fixed", "FR-027", "happy", nil, []combat.SetupOption{combat.WithInitiativeMode(combat.MonsterInitiativeFixed)}, fixed(17)},
		{"score above 10 plus modifier", "FR-027", "unhappy (rule interaction)", msInitiative(22, 7), []combat.SetupOption{combat.WithInitiativeMode(combat.MonsterInitiativeFixed)}, fixed(22)},
		{"negative modifier, score above", "FR-027", "unhappy (rule interaction)", msInitiative(13, -2), []combat.SetupOption{combat.WithInitiativeMode(combat.MonsterInitiativeFixed)}, fixed(13)},
		{"negative modifier, rolled", "FR-026", "unhappy (boundary)", msInitiative(13, -2), nil, rolled(-2)},
		{"default mode", "FR-025", "happy", msInitiative(12, 2), nil, rolled(2)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := msSetup(t, msMonster(t, c.edit), c.options...).Initiative(); got != c.want {
				t.Errorf("Initiative = %#v, want %#v [%s]", got, c.want, setupSpec+c.requirement)
			}
		})
	}
	if combat.MonsterInitiativeRolled.String() != "rolled" || combat.MonsterInitiativeFixed.String() != "fixed" || combat.MonsterInitiativeMode(0).String() != "unknown" ||
		combat.MonsterInitiativeMode(0).Valid() || combat.MonsterInitiativeMode(3).Valid() {
		t.Error("initiative mode names or validity changed (FR-025)")
	}
}

type modifierRow struct {
	ability, total int
	prof           int
	hasProf        bool
	others         []int
}

func rowOf(r d20.RollModifiers) modifierRow {
	var others []int
	for _, o := range r.Others() {
		others = append(others, o.Value())
	}
	p, has := r.ProficiencyBonus()
	return modifierRow{r.Ability().Value(), r.Total().Value(), p.Value(), has, others}
}

func (r modifierRow) equal(o modifierRow) bool {
	return r.ability == o.ability && r.total == o.total && r.prof == o.prof && r.hasProf == o.hasProf && slices.Equal(r.others, o.others)
}

// msModifiers is a Challenge Rating 1 monster (Proficiency Bonus +2) with every score 16 (+3) and the saves and skills that
// give each case of the split: a save equal to the ability modifier, one bonus above it, twice the bonus above it (a
// save has no doubled slot), another difference, a negative one, and skills at one bonus, twice the bonus and one below.
func msModifiers(f *content.MonsterFields) {
	f.ChallengeRating = content.MustChallengeRating("1")
	scores := map[ability.Ability]ability.Score{}
	saves := map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a] = ability.MustScore(16)
	}
	saves[ability.Strength], saves[ability.Dexterity], saves[ability.Constitution] = 6, 3, 5
	saves[ability.Intelligence], saves[ability.Wisdom], saves[ability.Charisma] = 9, 7, 2
	f.Abilities = luMust(content.NewAbilityBlock(luMust(ability.NewScores(scores)), saves))
	f.Skills = map[content.Skill]int{content.SkillStealth: 1, content.SkillPerception: 5, content.SkillAthletics: 7, content.SkillInsight: 8}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-030 to FR-032, table 4.6a - the modifiers of a save or a skill split the printed value into the ability modifier, the Proficiency Bonus (twice it for a skill only) and an authoritative remainder in the others, and sum to the printed value; a skill the block does not list and an ability check are the ability modifier alone (SRD 5.2.1, "Ability Scores", "Skills", pp. 255-256; "Expertise", p. 182).
// Path: per case.
func TestMonsterSetupModifiers(t *testing.T) {
	t.Parallel()
	save := func(a ability.Ability) func(combat.MonsterSetup) d20.RollModifiers {
		return func(s combat.MonsterSetup) d20.RollModifiers { return s.SavingThrow(a) }
	}
	skill := func(sk content.Skill) func(combat.MonsterSetup) d20.RollModifiers {
		return func(s combat.MonsterSetup) d20.RollModifiers { return s.SkillCheck(sk) }
	}
	cases := []struct {
		name, requirement, path string
		edit                    func(*content.MonsterFields)
		request                 func(combat.MonsterSetup) d20.RollModifiers
		want                    modifierRow
	}{
		{"save, not proficient", "FR-030", "happy", nil, save(ability.Strength), modifierRow{5, 5, 0, false, nil}},
		{"save, proficient", "FR-030", "happy", nil, save(ability.Dexterity), modifierRow{-1, 3, 4, true, nil}},
		{"skill, doubled", "FR-031", "unhappy (rule interaction)", nil, skill(content.SkillPerception), modifierRow{2, 10, 8, true, nil}},
		{"skill, doubled, other ability", "FR-031", "unhappy (rule interaction)", nil, skill(content.SkillHistory), modifierRow{4, 12, 8, true, nil}},
		{"skill not listed", "FR-031", "unhappy (empty and minimal)", nil, skill(content.SkillArcana), modifierRow{4, 4, 0, false, nil}},
		{"ability check", "FR-032", "happy", nil, func(s combat.MonsterSetup) d20.RollModifiers { return s.AbilityCheck(ability.Charisma) }, modifierRow{4, 4, 0, false, nil}},
		{"a difference that is neither", "FR-030", "unhappy (invalid input)", msModifiers, save(ability.Strength), modifierRow{3, 6, 0, false, []int{3}}},
		{"printed equal to the ability modifier", "FR-030", "happy", msModifiers, save(ability.Dexterity), modifierRow{3, 3, 0, false, nil}},
		{"one bonus above, a save", "FR-030", "unhappy (boundary)", msModifiers, save(ability.Constitution), modifierRow{3, 5, 2, true, nil}},
		{"twice the bonus above, a save has no doubled slot", "FR-030", "unhappy (rule interaction)", msModifiers, save(ability.Wisdom), modifierRow{3, 7, 0, false, []int{4}}},
		{"another difference above", "FR-030", "unhappy (boundary)", msModifiers, save(ability.Intelligence), modifierRow{3, 9, 0, false, []int{6}}},
		{"a negative difference, a save", "FR-030", "unhappy (boundary)", msModifiers, save(ability.Charisma), modifierRow{3, 2, 0, false, []int{-1}}},
		{"a negative difference, a skill", "FR-031", "unhappy (boundary)", msModifiers, skill(content.SkillStealth), modifierRow{3, 1, 0, false, []int{-2}}},
		{"a skill at one bonus", "FR-031", "unhappy (boundary)", msModifiers, skill(content.SkillPerception), modifierRow{3, 5, 2, true, nil}},
		{"a skill above twice the bonus", "FR-031", "unhappy (boundary)", msModifiers, skill(content.SkillInsight), modifierRow{3, 8, 0, false, []int{5}}},
		{"a skill at twice the bonus", "FR-031", "unhappy (boundary)", msModifiers, skill(content.SkillAthletics), modifierRow{3, 7, 4, true, nil}},
		{"an invalid ability", "FR-030", "unhappy (invalid input)", nil, save(ability.Ability(0)), modifierRow{}},
		{"an invalid skill", "FR-031", "unhappy (invalid input)", nil, skill(content.Skill(0)), modifierRow{}},
		{"an invalid ability check", "FR-032", "unhappy (invalid input)", nil, func(s combat.MonsterSetup) d20.RollModifiers { return s.AbilityCheck(ability.Ability(9)) }, modifierRow{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := rowOf(c.request(msSetup(t, msMonster(t, c.edit)))); !got.equal(c.want) {
				t.Errorf("modifiers = %+v, want %+v [%s]", got, c.want, setupSpec+c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-033 - for every ability and every listed skill of several blocks the slots sum to the printed value and hold at most one Proficiency Bonus.
// Path: unhappy (rule interaction).
func TestMonsterSetupModifiersSumToThePrintedValues(t *testing.T) {
	t.Parallel()
	for name, edit := range map[string]func(*content.MonsterFields){"first block": nil, "split cases": msModifiers} {
		m := msMonster(t, edit)
		s := msSetup(t, m)
		for _, a := range ability.All() {
			if got := s.SavingThrow(a).Total().Value(); got != m.Abilities().Save(a) {
				t.Errorf("%s: %v save total = %d, want the printed %d", name, a, got, m.Abilities().Save(a))
			}
			if got := s.AbilityCheck(a).Total().Value(); got != m.AbilityModifier(a).Value() {
				t.Errorf("%s: %v check total = %d, want the modifier %d", name, a, got, m.AbilityModifier(a).Value())
			}
		}
		for sk, printed := range m.Skills() {
			if got := s.SkillCheck(sk).Total().Value(); got != printed {
				t.Errorf("%s: %v total = %d, want the printed %d", name, sk, got, printed)
			}
		}
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-029, table 4.6a (bands) - the Proficiency Bonus is that of the Challenge Rating, +2 through +9 in the bands 0-4, 5-8, 9-12, 13-16, 17-20, 21-24, 25-28 and 29-30, for all 34 ratings, and goes past +6 (SRD 5.2.1, "Proficiency Bonus", p. 256).
// Path: unhappy (boundary).
func TestMonsterSetupProficiencyBonusByChallengeRating(t *testing.T) {
	t.Parallel()
	bands := map[string]int{"0": 2, "1/8": 2, "1/4": 2, "1/2": 2, "1": 2, "4": 2, "5": 3, "8": 3, "9": 4, "12": 4, "13": 5, "16": 5, "17": 6, "20": 6, "21": 7, "24": 7, "25": 8, "28": 8, "29": 9, "30": 9}
	for cr, want := range bands {
		s := msSetup(t, msMonster(t, func(f *content.MonsterFields) { f.ChallengeRating = content.MustChallengeRating(cr) }))
		if got := s.ProficiencyBonus().Value(); got != want {
			t.Errorf("Challenge Rating %s: Proficiency Bonus = %+d, want %+d", cr, got, want)
		}
	}
	all := content.AllChallengeRatings()
	if len(all) != 34 {
		t.Fatalf("%d Challenge Ratings, want 34", len(all))
	}
	for _, cr := range all {
		s := msSetup(t, msMonster(t, func(f *content.MonsterFields) { f.ChallengeRating = cr }))
		if s.ProficiencyBonus().Value() != s.Monster().ProficiencyBonus() {
			t.Errorf("Challenge Rating %s: setup bonus %v differs from the content bonus %d", cr, s.ProficiencyBonus(), s.Monster().ProficiencyBonus())
		}
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-030, plan-phase-4.md P13 and security (arithmetic) - a printed modifier whose difference from the ability modifier does not fit an int is out of range on a 32-bit int (ContentNumberOutOfRange) and splits exactly on a 64-bit one; a printed modifier of the largest magnitude whose difference fits is accepted.
// Path: unhappy (boundary).
func TestMonsterSetupModifierDifferenceAtTheIntLimit(t *testing.T) {
	t.Parallel()
	fits := strconv.IntSize == 64
	m := msMonster(t, func(f *content.MonsterFields) {
		scores := map[ability.Ability]ability.Score{}
		saves := map[ability.Ability]int{}
		for _, a := range ability.All() {
			scores[a], saves[a] = ability.MustScore(1), content.MonsterMaxNumber // modifier -5: the difference is 2^31 + 4
		}
		f.Abilities = luMust(content.NewAbilityBlock(luMust(ability.NewScores(scores)), saves))
	})
	s, err := combat.NewMonsterSetup(m, combat.PrintedAverageHitPoints{})
	if !fits {
		var rej *combat.MonsterSetupError
		if !errors.As(err, &rej) || !slices.Equal(rej.Problems(), []combat.MonsterSetupProblem{combat.ContentNumberOutOfRange}) {
			t.Errorf("err = %v, want ContentNumberOutOfRange at 32-bit width", err)
		}
		return
	}
	if err != nil || s.SavingThrow(ability.Strength).Total().Value() != content.MonsterMaxNumber {
		t.Errorf("a save of %d = %v, %v, want it kept exactly at 64-bit width", content.MonsterMaxNumber, s.SavingThrow(ability.Strength), err)
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-034, FR-035, table 4.7a - unqualified single-mode entries become the speeds in feet (the hover flag is not copied and stays on the monster); a qualified or multi-mode entry is unapplied in printed order; with no walk entry the walking Speed is 0 (SRD 5.2.1, "Speed", p. 255).
// Path: per case.
func TestMonsterSetupSpeeds(t *testing.T) {
	t.Parallel()
	fly := func(hover bool) content.SpeedEntry {
		return luMust(content.NewSpeedEntry([]content.SpeedMode{content.SpeedFly}, 90, hover, ""))
	}
	wolf := msSpeed(40, "wolf form only", content.SpeedWalk)
	climbOrFly := luMust(content.NewSpeedEntry([]content.SpeedMode{content.SpeedClimb, content.SpeedFly}, 20, false, "GM's choice"))
	plainTwoModes := luMust(content.NewSpeedEntry([]content.SpeedMode{content.SpeedBurrow, content.SpeedClimb}, 15, false, ""))
	speeds := func(es ...content.SpeedEntry) func(*content.MonsterFields) {
		return func(f *content.MonsterFields) { f.Speeds = es }
	}
	cases := []struct {
		name, requirement, path string
		edit                    func(*content.MonsterFields)
		walk                    int
		special                 map[combat.SpeedName]int
		unapplied               []content.SpeedEntry
	}{
		{"walk and swim", "FR-034", "happy", nil, 10, map[combat.SpeedName]int{combat.SpeedNameSwim: 40}, nil},
		{"hover not copied", "FR-034", "happy", speeds(msSpeed(10, "", content.SpeedWalk), fly(true)), 10, map[combat.SpeedName]int{combat.SpeedNameFly: 90}, nil},
		{"form qualifier", "FR-035", "unhappy (rule interaction)", speeds(msSpeed(30, "", content.SpeedWalk), wolf), 30, nil, []content.SpeedEntry{wolf}},
		{"alternative modes", "FR-035", "unhappy (rule interaction)", speeds(msSpeed(20, "", content.SpeedWalk), climbOrFly), 20, nil, []content.SpeedEntry{climbOrFly}},
		{"several modes without a qualifier", "FR-035", "unhappy (rule interaction)", speeds(msSpeed(20, "", content.SpeedWalk), plainTwoModes), 20, nil, []content.SpeedEntry{plainTwoModes}},
		{"unapplied in printed order", "FR-035", "unhappy (boundary)", speeds(climbOrFly, msSpeed(20, "", content.SpeedWalk), wolf), 20, nil, []content.SpeedEntry{climbOrFly, wolf}},
		{"no walk entry", "FR-034", "unhappy (boundary)", speeds(msSpeed(30, "", content.SpeedFly)), 0, map[combat.SpeedName]int{combat.SpeedNameFly: 30}, nil},
		{"zero Speed", "FR-034", "unhappy (boundary)", speeds(msSpeed(0, "", content.SpeedWalk)), 0, nil, nil},
		{"a special speed of 0 is had", "FR-034", "unhappy (boundary)", speeds(msSpeed(10, "", content.SpeedWalk), msSpeed(0, "", content.SpeedBurrow)), 10, map[combat.SpeedName]int{combat.SpeedNameBurrow: 0}, nil},
		{"every mode", "FR-034", "happy", speeds(msSpeed(1, "", content.SpeedWalk), msSpeed(2, "", content.SpeedBurrow), msSpeed(3, "", content.SpeedClimb), msSpeed(4, "", content.SpeedFly), msSpeed(5, "", content.SpeedSwim)), 1,
			map[combat.SpeedName]int{combat.SpeedNameBurrow: 2, combat.SpeedNameClimb: 3, combat.SpeedNameFly: 4, combat.SpeedNameSwim: 5}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := setupSpec + c.requirement
			s := msSetup(t, msMonster(t, c.edit))
			if s.Speeds().Walk() != c.walk {
				t.Errorf("walking Speed = %d, want %d [%s]", s.Speeds().Walk(), c.walk, req)
			}
			for _, n := range []combat.SpeedName{combat.SpeedNameBurrow, combat.SpeedNameClimb, combat.SpeedNameFly, combat.SpeedNameSwim} {
				feet, has := s.Speeds().Of(n)
				want, wantHas := c.special[n]
				if feet != want || has != wantHas {
					t.Errorf("%v = %d, %v, want %d, %v [%s]", n, feet, has, want, wantHas, req)
				}
			}
			var got []content.SpeedEntry
			for _, u := range s.UnappliedSpeeds() {
				got = append(got, u.Entry())
			}
			if !slices.EqualFunc(got, c.unapplied, content.SpeedEntry.Equal) {
				t.Errorf("unapplied speeds = %v, want %v [%s]", got, c.unapplied, req)
			}
		})
	}
	hover := msSetup(t, msMonster(t, speeds(msSpeed(10, "", content.SpeedWalk), fly(true))))
	if kept := hover.Monster().Speeds(); len(kept) != 2 || !kept[1].Hover() {
		t.Errorf("the monster's speeds = %v, want the fly entry with hover still readable", kept)
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-036 - Profile returns the id, the chosen size and the applied speeds.
// Path: happy.
func TestMonsterSetupProfile(t *testing.T) {
	t.Parallel()
	s := msSetup(t, msMonster(t, msTwoSizes), combat.WithSize(content.SizeSmall))
	id := combat.MustCombatantID("a")
	if got, want := s.Profile(id), (combat.CombatantProfile{ID: id, Size: content.SizeSmall, Speeds: s.Speeds()}); got != want {
		t.Errorf("Profile = %+v, want %+v", got, want)
	}
}
