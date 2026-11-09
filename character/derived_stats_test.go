package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// base returns a minimal valid calculator input: a level 1 Wizard with flat
// scores of 10 and a stored maximum of 10.
func base(t *testing.T) character.DerivedStatsInput {
	t.Helper()
	return character.DerivedStatsInput{
		ClassLevels:     levels(cl("wizard", 1)),
		AbilityScores:   flatScores(t, 10),
		HitPointMaximum: character.MustHitPointMaximum(10),
		InitialClass:    wizardClass(t),
	}
}

func calculate(t *testing.T, in character.DerivedStatsInput) character.DerivedStats {
	t.Helper()
	got, err := character.DerivedStatsCalculator{}.Calculate(in)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	c, ok := got.(character.DerivedStatsComputed)
	if !ok {
		t.Fatalf("Calculate = %#v, want DerivedStatsComputed", got)
	}
	// FR-067: every breakdown the calculator builds satisfies the constructor.
	check := func(name string, b character.ModifierBreakdown) {
		if _, err := character.NewModifierBreakdown(b.Terms()...); err != nil {
			t.Errorf("%s breakdown violates FR-067: %v", name, err)
		}
	}
	check("armor class", c.Stats.ArmorClassBreakdown())
	check("hit points", c.Stats.HitPointMaximumBreakdown())
	check("passive perception", c.Stats.PassivePerception().Breakdown())
	for _, s := range c.Stats.SavingThrows() {
		check("save", s.Breakdown())
	}
	for _, s := range c.Stats.Skills() {
		check("skill", s.Breakdown())
	}
	return c.Stats
}

func scoreForModifier(m int) int { return 10 + 2*m }

// Requirement: doc/specs/derived-stats.md FR-009, FR-059 - the Proficiency Bonus
// comes from the total level, not the level in one class (SRD 5.2.1,
// "Multiclassing", p. 25).
// Path: per case.
func TestProficiencyBonusFromTotalLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		levels []character.ClassLevel
		want   int
	}{
		{"3 Fighter / 2 Rogue is total 5", []character.ClassLevel{cl("fighter", 3), cl("rogue", 2)}, 3},
		{"4 Cleric / 4 Fighter is total 8", []character.ClassLevel{cl("cleric", 4), cl("fighter", 4)}, 3},
		{"level 1", []character.ClassLevel{cl("wizard", 1)}, 2},
		{"level 20", []character.ClassLevel{cl("wizard", 20)}, 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base(t)
			in.ClassLevels = character.MustClassLevels(c.levels)
			s := calculate(t, in)
			if s.ProficiencyBonus().Value() != c.want || s.TotalLevel() != in.ClassLevels.TotalLevel() {
				t.Errorf("proficiency bonus = %d, total level %d, want bonus %d", s.ProficiencyBonus().Value(), s.TotalLevel().Value(), c.want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-012 to FR-015, FR-010 - one
// modifier for each of the 18 skills: ability modifier, plus the Proficiency
// Bonus when proficient, plus twice it with Expertise, in one proficiency term
// (SRD 5.2.1, "Skill Proficiencies", p. 8; "Expertise", p. 182).
// Path: per case.
func TestSkillModifiers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		profs      character.SkillProficiencies
		level      int
		want       int
		source     character.DerivedStatSource // the proficiency term, 0 for none
	}{
		{"proficient", "happy", character.MustSkillProficiencies([]content.Skill{content.SkillAthletics}, nil), 1, 5, character.ProficiencyBonusTerm},
		{"not proficient", "unhappy (rule interaction)", character.SkillProficiencies{}, 1, 3, 0},
		{"Expertise at level 1", "happy", character.MustSkillProficiencies([]content.Skill{content.SkillAthletics}, []content.Skill{content.SkillAthletics}), 1, 7, character.ExpertiseTerm},
		{"proficient at level 5", "unhappy (boundary)", character.MustSkillProficiencies([]content.Skill{content.SkillAthletics}, nil), 5, 6, character.ProficiencyBonusTerm},
		{"Expertise at level 20", "unhappy (boundary)", character.MustSkillProficiencies([]content.Skill{content.SkillAthletics}, []content.Skill{content.SkillAthletics}), 20, 15, character.ExpertiseTerm},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base(t)
			in.ClassLevels = levels(cl("wizard", c.level))
			in.AbilityScores = scoresOf(t, 16, 10, 10, 10, 10, 10)
			in.SkillProficiencies = c.profs
			s := calculate(t, in)
			m := s.Skill(content.SkillAthletics)
			if m.Value() != c.want || m.Skill() != content.SkillAthletics {
				t.Errorf("Athletics = %d, want %d", m.Value(), c.want)
			}
			proficiencyTerms := 0
			for _, term := range m.Breakdown().Terms() {
				if term.Source == character.ProficiencyBonusTerm || term.Source == character.ExpertiseTerm {
					proficiencyTerms++
					if term.Source != c.source {
						t.Errorf("proficiency term source = %v, want %v", term.Source, c.source)
					}
				}
			}
			if (c.source == 0) != (proficiencyTerms == 0) || proficiencyTerms > 1 {
				t.Errorf("%d proficiency terms for source %v", proficiencyTerms, c.source)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-012, FR-042 - exactly 18 skill and 6
// saving throw modifiers whatever the proficiencies, each skill with its own
// ability; an all-proficient character still has one proficiency term each.
// Path: unhappy (empty and minimal).
func TestEveryModifierPresent(t *testing.T) {
	t.Parallel()
	var all []content.Skill
	for s := content.SkillAcrobatics; s <= content.SkillSurvival; s++ {
		all = append(all, s)
	}
	for name, p := range map[string]character.SkillProficiencies{"none": {}, "all": character.MustSkillProficiencies(all, all)} {
		in := base(t)
		in.SkillProficiencies = p
		s := calculate(t, in)
		skills, saves := s.Skills(), s.SavingThrows()
		if len(skills) != 18 || len(saves) != 6 {
			t.Fatalf("%s: %d skills, %d saves", name, len(skills), len(saves))
		}
		for i, m := range skills {
			if m.Skill() != content.SkillAcrobatics+content.Skill(i) {
				t.Errorf("%s: skill %d is %v", name, i, m.Skill())
			}
		}
		for i, m := range saves {
			if m.Ability() != ability.All()[i] || s.SavingThrow(m.Ability()).Ability() != m.Ability() {
				t.Errorf("%s: save %d is %v", name, i, m.Ability())
			}
		}
		if s.Skill(0).Value() != 0 || s.SavingThrow(0).Value() != 0 {
			t.Errorf("%s: an invalid skill or ability has a modifier", name)
		}
	}
}

// Requirement: doc/specs/derived-stats.md FR-017 - a save is the ability
// modifier plus the Proficiency Bonus when proficient (SRD 5.2.1, "Saving
// Throws", p. 7).
// Path: per case.
func TestSavingThrowModifiers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		con, level int
		fighter    bool
		want       int
	}{
		{"proficient save", "happy", 2, 1, true, 4},
		{"not proficient", "unhappy (rule interaction)", 2, 1, false, 2},
		{"proficient, negative ability", "unhappy (boundary)", -1, 5, true, 2},
		{"not proficient, negative ability", "unhappy (boundary)", -1, 5, false, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base(t)
			in.ClassLevels = levels(cl("wizard", c.level))
			in.AbilityScores = scoresOf(t, 10, 10, scoreForModifier(c.con), 10, 10, 10)
			if c.fighter {
				in.ClassLevels = levels(cl("fighter", c.level))
				in.InitialClass = fighterClass(t)
			}
			if got := calculate(t, in).SavingThrow(ability.Constitution).Value(); got != c.want {
				t.Errorf("Constitution save = %d, want %d", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-049, FR-019 - saving throw
// proficiencies are the initial class's plus additionalSavingThrows; later
// classes add none (SRD 5.2.1, "Multiclassing", p. 25; "Slippery Mind", p. 63).
// Path: per case.
func TestSavingThrowProficienciesFromInitialClass(t *testing.T) {
	t.Parallel()
	rogue := func(t *testing.T) content.ClassDefinition {
		t.Helper()
		return classFixture(t, "rogue", "D8", []string{"dexterity", "intelligence"}, nil)
	}
	cases := []struct {
		name, path string
		levels     []character.ClassLevel
		initial    func(*testing.T) content.ClassDefinition
		extra      []ability.Ability
		want       []ability.Ability
	}{
		{"Fighter 5 / Rogue 15 with Slippery Mind", "unhappy (rule interaction)", []character.ClassLevel{cl("fighter", 5), cl("rogue", 15)}, fighterClass,
			[]ability.Ability{ability.Wisdom, ability.Charisma}, []ability.Ability{ability.Strength, ability.Constitution, ability.Wisdom, ability.Charisma}},
		{"Fighter 5 / Rogue 14", "unhappy (boundary)", []character.ClassLevel{cl("fighter", 5), cl("rogue", 14)}, fighterClass,
			nil, []ability.Ability{ability.Strength, ability.Constitution}},
		{"Rogue 15 first", "happy", []character.ClassLevel{cl("rogue", 15)}, rogue,
			[]ability.Ability{ability.Wisdom, ability.Charisma}, []ability.Ability{ability.Dexterity, ability.Intelligence, ability.Wisdom, ability.Charisma}},
		{"Wizard initial plus a Fighter level", "unhappy (rule interaction)", []character.ClassLevel{cl("wizard", 1), cl("fighter", 1)}, wizardClass,
			nil, []ability.Ability{ability.Intelligence, ability.Wisdom}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base(t)
			in.ClassLevels = character.MustClassLevels(c.levels)
			in.InitialClass = c.initial(t)
			in.AdditionalSavingThrows = must(character.NewSavingThrowProficiencies(c.extra...))
			s := calculate(t, in)
			var got []ability.Ability
			for _, m := range s.SavingThrows() {
				for _, term := range m.Breakdown().Terms() {
					if term.Source == character.ProficiencyBonusTerm {
						got = append(got, m.Ability())
					}
				}
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("proficient saves = %v, want %v", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-079 - a flat save bonus is added
// after the proficiency term as one flatBonus term to every save it covers, and
// to nothing else (SRD 5.2.1, "Ring of Protection", p. 238).
// Path: per case.
func TestFlatSavingThrowBonuses(t *testing.T) {
	t.Parallel()
	in := base(t)
	in.InitialClass = fighterClass(t)
	in.ClassLevels = levels(cl("fighter", 1))
	in.AbilityScores = scoresOf(t, 8, 10, 14, 10, 10, 10)
	in.FlatSavingThrowBonuses = []character.FlatSavingThrowBonus{must(character.NewFlatSavingThrowBonus(itemBonus(t, 1), nil))}
	s := calculate(t, in)
	if got := s.SavingThrow(ability.Constitution).Value(); got != 5 {
		t.Errorf("Constitution save = %d, want 5", got)
	}
	if got := s.SavingThrow(ability.Dexterity).Value(); got != 1 {
		t.Errorf("Dexterity save = %d, want 1", got)
	}
	if got := s.Skill(content.SkillAthletics).Value(); got != -1 {
		t.Errorf("skill gained a save bonus: %d", got)
	}
	terms := s.SavingThrow(ability.Constitution).Breakdown().Terms()
	if last := terms[len(terms)-1]; last.Source != character.FlatBonusTerm || last.Origin == nil {
		t.Errorf("last Constitution term = %+v, want the flatBonus", last)
	}

	in.FlatSavingThrowBonuses = []character.FlatSavingThrowBonus{must(character.NewFlatSavingThrowBonus(itemBonus(t, 1), []ability.Ability{ability.Dexterity}))}
	s = calculate(t, in)
	for _, m := range s.SavingThrows() {
		want := in.AbilityScores.Modifier(m.Ability()).Value()
		if m.Ability() == ability.Strength || m.Ability() == ability.Constitution {
			want += 2
		}
		if m.Ability() == ability.Dexterity {
			want++
		}
		if m.Value() != want {
			t.Errorf("%v save = %d, want %d", m.Ability(), m.Value(), want)
		}
	}
}

// Requirement: doc/specs/derived-stats.md FR-020, FR-021 - Passive Perception is
// 10 plus the Perception skill modifier, plus 5 with Advantage and minus 5 with
// Disadvantage, as a rollMode term (SRD 5.2.1, "Passive Perception", pp. 22,
// 186).
// Path: per case.
func TestPassivePerception(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		wis        int
		proficient bool
		mode       modifier.RollMode
		want       int
	}{
		{"SRD example", "happy", 15, true, modifier.Normal, 14},
		{"SRD example with Advantage", "unhappy (rule interaction)", 15, true, modifier.WithAdvantage, 19},
		{"Disadvantage", "unhappy (rule interaction)", 15, true, modifier.WithDisadvantage, 9},
		{"lowest Wisdom", "unhappy (boundary)", 1, false, modifier.Normal, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base(t)
			in.AbilityScores = scoresOf(t, 10, 10, 10, 10, c.wis, 10)
			in.PerceptionRollMode = c.mode
			if c.proficient {
				in.SkillProficiencies = character.MustSkillProficiencies([]content.Skill{content.SkillPerception}, nil)
			}
			pp := calculate(t, in).PassivePerception()
			if pp.Value() != c.want {
				t.Errorf("Passive Perception = %d, want %d", pp.Value(), c.want)
			}
			terms := pp.Breakdown().Terms()
			if terms[0].Source != character.BaseValueTerm || terms[0].Modifier.Value() != 10 {
				t.Errorf("first term = %+v, want baseValue 10", terms[0])
			}
			if hasRollMode := terms[len(terms)-1].Source == character.RollModeTerm; hasRollMode != (c.mode != modifier.Normal) {
				t.Errorf("rollMode term present = %v for mode %v", hasRollMode, c.mode)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-050, FR-051, FR-029 - the reported
// maximum is the stored value plus each bonus contribution, shown as one term per
// bonus, zero contributions included (SRD 5.2.1, "Draconic Resilience", p. 69).
// Path: per case.
func TestHitPointMaximumBonuses(t *testing.T) {
	t.Parallel()
	draconic := character.MustHitPointMaximumBonus(3, 1, 3, "sorcerer")
	cases := []struct {
		name, path string
		levels     []character.ClassLevel
		bonuses    []character.HitPointMaximumBonus
		stored     int
		want       int
		terms      int
	}{
		{"no bonuses", "happy", []character.ClassLevel{cl("sorcerer", 3)}, nil, 20, 20, 1},
		{"Draconic at level 3", "happy", []character.ClassLevel{cl("sorcerer", 3)}, []character.HitPointMaximumBonus{draconic}, 20, 23, 2},
		{"Draconic at level 4", "unhappy (boundary)", []character.ClassLevel{cl("sorcerer", 4)}, []character.HitPointMaximumBonus{draconic}, 26, 30, 2},
		{"Draconic below level 3", "unhappy (boundary)", []character.ClassLevel{cl("sorcerer", 2)}, []character.HitPointMaximumBonus{draconic}, 14, 14, 2},
		{"Draconic scoped to Sorcerer", "unhappy (rule interaction)", []character.ClassLevel{cl("sorcerer", 3), cl("wizard", 2)}, []character.HitPointMaximumBonus{draconic}, 30, 33, 2},
		{"Dwarven Toughness level 1", "happy", []character.ClassLevel{cl("fighter", 1)}, []character.HitPointMaximumBonus{character.MustHitPointMaximumBonus(1, 1, 1, "")}, 12, 13, 2},
		{"two bonuses", "unhappy (rule interaction)", []character.ClassLevel{cl("sorcerer", 4)}, []character.HitPointMaximumBonus{draconic, character.MustHitPointMaximumBonus(1, 1, 1, "")}, 26, 34, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base(t)
			in.ClassLevels = character.MustClassLevels(c.levels)
			in.HitPointMaximum = character.MustHitPointMaximum(c.stored)
			in.HitPointMaximumBonuses = c.bonuses
			s := calculate(t, in)
			if s.HitPointMaximum().Value() != c.want || len(s.HitPointMaximumBreakdown().Terms()) != c.terms {
				t.Errorf("maximum = %d with %d terms, want %d with %d", s.HitPointMaximum().Value(), len(s.HitPointMaximumBreakdown().Terms()), c.want, c.terms)
			}
			if first := s.HitPointMaximumBreakdown().Terms()[0]; first.Source != character.StoredHitPointMaximumTerm || first.Modifier.Value() != c.stored {
				t.Errorf("first term = %+v, want the stored maximum", first)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-028 - the Hit Dice pool in the
// result is the command's, unchanged.
// Path: happy.
func TestHitDicePoolPassesThrough(t *testing.T) {
	t.Parallel()
	in := base(t)
	in.HitDicePool = character.MustHitDicePoolFromLevels(levels(cl("wizard", 1)), dieMap())
	if got := calculate(t, in).HitDicePool(); !got.Equal(in.HitDicePool) || got.Total() != 1 {
		t.Errorf("pool = %v", got.Entries())
	}
}

// Requirement: doc/specs/derived-stats.md FR-042 - a level 1 character with the
// lowest valid inputs gets a complete result.
// Path: unhappy (empty and minimal).
func TestMinimalCharacter(t *testing.T) {
	t.Parallel()
	in := base(t)
	in.AbilityScores = flatScores(t, 1)
	in.HitPointMaximum = character.MustHitPointMaximum(1)
	s := calculate(t, in)
	if s.HitPointMaximum().Value() != 1 || s.ArmorClass() != d20.ArmorClass(5) || s.PassivePerception().Value() != 5 ||
		len(s.Skills()) != 18 || len(s.SavingThrows()) != 6 || len(s.ArmorTrainingGaps()) != 0 {
		t.Errorf("minimal stats: HP %d AC %d PP %d", s.HitPointMaximum().Value(), s.ArmorClass(), s.PassivePerception().Value())
	}
}

// Requirement: doc/specs/derived-stats.md FR-045 - the result is identical on
// every call, the input is not altered and exposed collections are copies.
// Path: unhappy (determinism, immutability).
func TestCalculateDeterministicAndCopies(t *testing.T) {
	t.Parallel()
	in := base(t)
	in.FlatArmorClassBonuses = []character.FlatBonus{itemBonus(t, 1)}
	a, b := calculate(t, in), calculate(t, in)
	if !a.ArmorClassBreakdown().Equal(b.ArmorClassBreakdown()) || a.PassivePerception().Value() != b.PassivePerception().Value() {
		t.Error("two calls differ")
	}
	if len(in.FlatArmorClassBonuses) != 1 {
		t.Error("input altered")
	}
	skills := a.Skills()
	skills[0] = character.SkillModifier{}
	saves := a.SavingThrows()
	saves[0] = character.SavingThrowModifier{}
	if a.Skills()[0].Value() != b.Skills()[0].Value() || a.Skills()[0].Skill() == 0 || a.SavingThrows()[0].Ability() == 0 {
		t.Error("Skills or SavingThrows expose their slice")
	}
}

// Requirement: doc/specs/derived-stats.md FR-076, FR-080 - invalid inputs are
// rejected with ErrInvalidInput and no calculation.
// Path: per case.
func TestCalculateRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*testing.T, *character.DerivedStatsInput)
	}{
		{"no class levels", func(_ *testing.T, in *character.DerivedStatsInput) { in.ClassLevels = character.ClassLevels{} }},
		{"zero ability scores", func(_ *testing.T, in *character.DerivedStatsInput) { in.AbilityScores = ability.Scores{} }},
		{"zero Hit Point maximum", func(_ *testing.T, in *character.DerivedStatsInput) { in.HitPointMaximum = character.HitPointMaximum{} }},
		{"negative floor", func(_ *testing.T, in *character.DerivedStatsInput) { in.ArmorClassFloor = -1 }},
		{"unknown roll mode", func(_ *testing.T, in *character.DerivedStatsInput) { in.PerceptionRollMode = 9 }},
		{"zero flat bonus", func(_ *testing.T, in *character.DerivedStatsInput) {
			in.FlatArmorClassBonuses = []character.FlatBonus{{}}
		}},
		{"zero flat save bonus", func(_ *testing.T, in *character.DerivedStatsInput) {
			in.FlatSavingThrowBonuses = []character.FlatSavingThrowBonus{{}}
		}},
		{"zero Hit Point maximum bonus", func(_ *testing.T, in *character.DerivedStatsInput) {
			in.HitPointMaximumBonuses = []character.HitPointMaximumBonus{{}}
		}},
		{"zero alternative basis", func(_ *testing.T, in *character.DerivedStatsInput) {
			in.AlternativeBases = []character.AbilityBasedArmorClassBasis{{}}
		}},
		{"duplicate alternative ids", func(t *testing.T, in *character.DerivedStatsInput) {
			t.Helper()
			in.AlternativeBases = []character.AbilityBasedArmorClassBasis{basisOf(t, "barbarian"), basisOf(t, "barbarian")}
		}},
		{"unknown basis choice", func(_ *testing.T, in *character.DerivedStatsInput) { in.ChosenBasis = foreignChoice{} }},
		{"zero initial class", func(_ *testing.T, in *character.DerivedStatsInput) { in.InitialClass = content.ClassDefinition{} }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base(t)
			c.mutate(t, &in)
			got, err := character.DerivedStatsCalculator{}.Calculate(in)
			wantInvalid(t, err, c.name)
			if got != nil {
				t.Errorf("calculation %#v returned with an error", got)
			}
		})
	}
}

// foreignChoice is not one of the three ArmorClassBasisChoice variants; the
// marker method is unexported, so it embeds a real one to satisfy the interface.
type foreignChoice struct{ character.ChooseStandard }

// Requirement: doc/specs/derived-stats.md FR-063, FR-064 - the passed-through
// accessors: StealthCircumstances is the equipment rule unchanged.
// Path: per case.
func TestDerivedStatsStealthCircumstances(t *testing.T) {
	t.Parallel()
	s := calculate(t, base(t))
	cases := []struct {
		name, path string
		worn       []content.Armor
		want       int
	}{
		{"Stealth Disadvantage armor", "unhappy (rule interaction)", []content.Armor{chainMail(t)}, 1},
		{"no Stealth Disadvantage", "unhappy (rule interaction)", []content.Armor{hideArmor(t)}, 0},
		{"Shield only", "unhappy (empty and minimal)", []content.Armor{shieldOf(t)}, 0},
		{"nothing worn", "unhappy (empty and minimal)", nil, 0},
	}
	for _, c := range cases {
		if got := s.StealthCircumstances(c.worn); len(got) != c.want || (c.want == 1 && got[0] != modifier.Disadvantage{}) {
			t.Errorf("%s: StealthCircumstances = %v, want %d Disadvantage", c.name, got, c.want)
		}
	}
}
