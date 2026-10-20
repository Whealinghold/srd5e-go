package content_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

const monSpec = "doc/specs/content-monsters.md"

// The fixtures are synthetic: made-up names and numbers, never stat blocks.

// must returns v, and panics with err when it is not nil: a fixture that does not build is a bug in the test.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func monScores(v int) ability.Scores {
	m := map[ability.Ability]ability.Score{}
	for _, a := range ability.All() {
		m[a] = ability.MustScore(v)
	}
	return must(ability.NewScores(m))
}

func monSaves(v int) map[ability.Ability]int {
	m := map[ability.Ability]int{}
	for _, a := range ability.All() {
		m[a] = v
	}
	return m
}

func monAbilities(tb testing.TB) content.AbilityBlock {
	tb.Helper()
	return must(content.NewAbilityBlock(monScores(10), monSaves(0)))
}

func monHP(tb testing.TB) content.HitPointsEntry {
	tb.Helper()
	return must(content.NewHitPointsEntry(9, dice.MustExpression(2, dice.D8(), 0)))
}

func monSource() content.SourceCitation {
	return content.MustSourceCitation("Fixture", content.MonsterSection, 258)
}

func monSpeed(tb testing.TB, feet int) content.SpeedEntry {
	tb.Helper()
	return must(content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, feet, false, ""))
}

// monFields is the smallest valid stat block: one size, type, alignment, AC,
// Initiative, Hit Points, one Speed, six abilities, Passive Perception only,
// no Languages, CR 0 and no parts.
func monFields(tb testing.TB) content.MonsterFields {
	tb.Helper()
	return content.MonsterFields{
		Sizes:      []content.CreatureSize{content.SizeMedium},
		Type:       content.CreatureTypeBeast,
		Alignment:  content.AlignmentUnaligned,
		ArmorClass: 12,
		Initiative: must(content.NewStatBlockInitiative(1, 11)),
		HitPoints:  monHP(tb),
		Speeds:     []content.SpeedEntry{monSpeed(tb, 30)},
		Abilities:  monAbilities(tb), PassivePerception: 10,
	}
}

func monPart(tb testing.TB, name string, f content.StatBlockPartFields) content.StatBlockPart {
	tb.Helper()
	if f.Usage == nil {
		f.Usage = content.PartUnlimited{}
	}
	if f.Text == "" {
		f.Text = "T."
	}
	return must(content.NewStatBlockPart(name, f))
}

func monBuild(tb testing.TB, f content.MonsterFields) (content.Monster, error) {
	tb.Helper()
	return content.NewMonster("fixture_beast", "Fixture Beast", monSource(), f)
}

func monMake(tb testing.TB, f content.MonsterFields) content.Monster {
	tb.Helper()
	return must(monBuild(tb, f))
}

func panicsWithInvalidInput(f func()) (ok bool) {
	defer func() {
		r := recover()
		err, isErr := r.(error)
		ok = isErr && isInvalidInput(err)
	}()
	f()
	return false
}

// Requirement: monSpec FR-002, FR-093, table 4.1a - the id is the name lower-cased, apostrophes removed, runs of
// other characters one underscore, outer underscores trimmed; a name that derives to nothing is an *IDError.
// Path: per case.
func TestDeriveMonsterID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		in, want                string
	}{
		{"single word", "table 4.1a", "happy", "Aboleth", "aboleth"},
		{"spaces", "table 4.1a", "happy", "Adult Red Dragon", "adult_red_dragon"},
		{"hyphen becomes underscore", "table 4.1a", "happy", "Saber-Toothed Tiger", "saber_toothed_tiger"},
		{"hyphen in name", "table 4.1a", "happy", "Half-Dragon", "half_dragon"},
		{"curly apostrophe removed", "table 4.1a", "unhappy (boundary)", "Will-o’-Wisp", "will_o_wisp"},
		{"straight apostrophe removed", "table 4.1a", "unhappy (boundary)", "Will-o'-Wisp", "will_o_wisp"},
		{"repeated separators collapse", "table 4.1a", "unhappy (boundary)", "Giant  Rat", "giant_rat"},
		{"non-ASCII letter is a separator", "table 4.1a", "unhappy (boundary)", "Zoé", "zo"},
		{"outer separators trimmed", "FR-002", "unhappy (boundary)", " -Fixture Beast- ", "fixture_beast"},
		{"digits kept", "FR-002", "happy", "Beast 2", "beast_2"},
		{"Kelvin sign is not a k", "FR-002", "unhappy (boundary)", "Koi", "oi"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.DeriveMonsterID(c.in)
			if err != nil || got != content.MonsterID(c.want) || got != content.MustDeriveMonsterID(c.in) {
				t.Errorf("DeriveMonsterID(%q) = %q, %v, want %q [%s, %s]", c.in, got, err, c.want, c.requirement, c.path)
			}
		})
	}
	for _, in := range []string{"' - '", "", "2 Heads", "é", strings64()} {
		_, err := content.DeriveMonsterID(in)
		var idErr *content.IDError
		if !isInvalidInput(err) || !errors.As(err, &idErr) {
			t.Errorf("DeriveMonsterID(%q) error = %v, want an *IDError matching ErrInvalidInput [FR-093]", in, err)
		}
		if !panicsWithInvalidInput(func() { content.MustDeriveMonsterID(in) }) {
			t.Errorf("MustDeriveMonsterID(%q) did not panic with the error [FR-093]", in)
		}
	}
	if _, err := content.DeriveMonsterID("' - '"); !errors.Is(err, &content.IDError{Problem: content.IDEmpty}) {
		t.Errorf("no letters or digits: error = %v, want IDEmpty as cause [FR-093]", err)
	}
}

func strings64() string {
	b := make([]byte, 65)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

// Requirement: monSpec FR-030, FR-097, table 4.5a - exactly the 34 spellings parse, round trip through String,
// order totally and carry the Proficiency Bonus of their row; anything else is an *ParseError with the input.
// Path: per case.
func TestChallengeRating(t *testing.T) {
	t.Parallel()
	all := content.AllChallengeRatings()
	if len(all) != 34 {
		t.Fatalf("AllChallengeRatings has %d members, want 34 [FR-030]", len(all))
	}
	spellings := []string{"0", "1/8", "1/4", "1/2"}
	for n := 1; n <= 30; n++ {
		spellings = append(spellings, itoa(n))
	}
	for i, s := range spellings {
		cr, err := content.ParseChallengeRating(s)
		if err != nil || cr.String() != s || cr != all[i] || content.MustChallengeRating(s) != cr {
			t.Errorf("ParseChallengeRating(%q) = %v, %v, want member %d [FR-030]", s, cr, err, i)
		}
		if cr.IsZero() != (i == 0) {
			t.Errorf("%q.IsZero() = %v [FR-097]", s, cr.IsZero())
		}
		for j, o := range spellings {
			want := 0
			switch {
			case i < j:
				want = -1
			case i > j:
				want = 1
			}
			if got := content.CompareChallengeRating(cr, all[j]); got != want {
				t.Errorf("Compare(%q, %q) = %d, want %d [FR-097]", s, o, got, want)
			}
		}
	}
	bands := []struct {
		cr string
		pb int
	}{
		{"0", 2}, {"1/8", 2}, {"1/4", 2}, {"1/2", 2}, {"1", 2}, {"4", 2}, {"5", 3}, {"8", 3}, {"9", 4}, {"12", 4},
		{"13", 5}, {"16", 5}, {"17", 6}, {"20", 6}, {"21", 7}, {"24", 7}, {"25", 8}, {"28", 8}, {"29", 9}, {"30", 9},
	}
	for _, b := range bands {
		if got := content.MustChallengeRating(b.cr).ProficiencyBonus(); got != b.pb {
			t.Errorf("CR %s Proficiency Bonus = %d, want %d [FR-031, band edge]", b.cr, got, b.pb)
		}
	}
	for _, s := range []string{"1/3", "31", "-1", "0.5", "01", "", " 1", "1 ", "+1", "1/2 ", "00", "100", "1e1", "١", "1/16", "2/1"} {
		_, err := content.ParseChallengeRating(s)
		var pe *content.ParseError
		if !isInvalidInput(err) || !errors.As(err, &pe) || pe.Value != s {
			t.Errorf("ParseChallengeRating(%q) error = %v, want a *ParseError carrying the input [FR-030]", s, err)
		}
		if !panicsWithInvalidInput(func() { content.MustChallengeRating(s) }) {
			t.Errorf("MustChallengeRating(%q) did not panic with the error [FR-030]", s)
		}
	}
	if (content.ChallengeRating{}) != content.MustChallengeRating("0") {
		t.Error("the zero ChallengeRating is not CR 0 [FR-097]")
	}
}

// Requirement: monSpec FR-030 and doc/rules/testing.md section 10 - ParseChallengeRating never panics, and an
// accepted input is one of the 34 spellings that round-trips.
// Path: unhappy (invalid input).
func FuzzParseChallengeRating(f *testing.F) {
	for _, s := range []string{"0", "1/8", "30", "31", "", "-1", "01", "1/3", "0.5", "1e1", "١", " 1"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		cr, err := content.ParseChallengeRating(in)
		if err != nil {
			if !isInvalidInput(err) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if cr.String() != in {
			t.Fatalf("accepted %q but String() = %q", in, cr.String())
		}
	})
}

// Requirement: monSpec FR-001, FR-002, FR-003, FR-004 - identity: the id must be the derived one, the name a clean
// name, the source a page of 258 to 364 in one of the two sections.
// Path: per case.
func TestNewMonsterIdentity(t *testing.T) {
	t.Parallel()
	good := monSource()
	cases := []struct {
		name, requirement, path string
		id, monName             string
		source                  content.SourceCitation
		wantOK                  bool
	}{
		{"valid", "FR-001", "happy", "fixture_beast", "Fixture Beast", good, true},
		{"id differs from the name", "FR-002", "unhappy (invalid input)", "giant_rat", "Giant Bat", good, false},
		{"invalid id", "FR-002", "unhappy (invalid input)", "Fixture", "Fixture Beast", good, false},
		{"empty name", "FR-003", "unhappy (invalid input)", "fixture_beast", "", good, false},
		{"blank name", "FR-003", "unhappy (invalid input)", "fixture_beast", " ", good, false},
		{"padded name", "FR-003", "unhappy (invalid input)", "fixture_beast", " Fixture Beast", good, false},
		{"zero source", "FR-004", "unhappy (invalid input)", "fixture_beast", "Fixture Beast", content.SourceCitation{}, false},
		{"page 257", "FR-004", "unhappy (boundary)", "fixture_beast", "Fixture Beast", content.MustSourceCitation("F", content.MonsterSection, 257), false},
		{"page 258", "FR-004", "unhappy (boundary)", "fixture_beast", "Fixture Beast", content.MustSourceCitation("F", content.MonsterSection, 258), true},
		{"page 364", "FR-004", "unhappy (boundary)", "fixture_beast", "Fixture Beast", content.MustSourceCitation("F", content.MonsterAnimalsSection, 364), true},
		{"page 365", "FR-004", "unhappy (boundary)", "fixture_beast", "Fixture Beast", content.MustSourceCitation("F", content.MonsterAnimalsSection, 365), false},
		{"other section", "FR-004", "unhappy (invalid input)", "fixture_beast", "Fixture Beast", content.MustSourceCitation("F", "Equipment", 300), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			m, err := content.NewMonster(content.MonsterID(c.id), c.monName, c.source, monFields(t))
			if (err == nil) != c.wantOK || (err != nil && !isInvalidInput(err)) {
				t.Fatalf("NewMonster error = %v, want ok=%v [%s, %s]", err, c.wantOK, c.requirement, c.path)
			}
			if c.wantOK && (m.Key() != content.MonsterID(c.id).Key() || m.ID() != content.MonsterID(c.id) || m.Name() != c.monName || m.Source() != c.source) {
				t.Errorf("accessors = %v %q %q %v [FR-001]", m.Key(), m.ID(), m.Name(), m.Source())
			}
			if panicked := panicsWithInvalidInput(func() { content.MustMonster(content.MonsterID(c.id), c.monName, c.source, monFields(t)) }); panicked == c.wantOK {
				t.Errorf("MustMonster panicked = %v, want %v", panicked, !c.wantOK)
			}
		})
	}
	m := monMake(t, monFields(t))
	if e, ok := any(m).(content.Entry); !ok || e.Key().Kind() != content.KindMonster {
		t.Error("Monster is not an Entry of kind monster [FR-001]")
	}
}

// Requirement: monSpec FR-005, FR-006, FR-031 - a monster is immutable (inputs are copied, returned collections are
// copies), equal fields give equal monsters and changing one field makes them unequal; the Proficiency Bonus and
// ability modifier are derived.
// Path: per case.
func TestMonsterImmutabilityAndEquality(t *testing.T) {
	t.Parallel()
	rich := func(tb testing.TB) content.MonsterFields {
		tb.Helper()
		f := monFields(tb)
		f.Tags = []string{"Tag"}
		f.Skills = map[content.Skill]int{content.SkillStealth: 4}
		f.Resistances = []content.DefenseEntry{must(content.NewDamageTypeDefense(content.DamageCold, ""))}
		f.Gear = []string{"Item"}
		f.Senses = []content.SenseEntry{must(content.NewSenseEntry(content.SenseDarkvision, 60, ""))}
		f.Languages = must(must(content.NewLanguagesEntry([]string{"Common"})).WithTelepathy(30, ""))
		f.Traits = []content.StatBlockPart{monPart(tb, "Trait", content.StatBlockPartFields{})}
		f.Actions = []content.StatBlockPart{monPart(tb, "Bite", content.StatBlockPartFields{})}
		f.LegendaryActions = must(content.NewLegendaryActions(3, 0, []content.StatBlockPart{monPart(tb, "Move", content.StatBlockPartFields{})}))
		f.ChallengeRating = content.MustChallengeRating("5")
		f.ExperiencePoints, f.ExperiencePointsInLair = 100, 200
		return f
	}
	base := monMake(t, rich(t))
	if !base.Equal(monMake(t, rich(t))) {
		t.Error("equal fields give unequal monsters [FR-006]")
	}
	mutators := map[string]func(*content.MonsterFields){
		"sizes":     func(f *content.MonsterFields) { f.Sizes = []content.CreatureSize{content.SizeLarge} },
		"swarm":     func(f *content.MonsterFields) { f.Swarm = must(content.NewSwarmComposition(content.SizeTiny)) },
		"type":      func(f *content.MonsterFields) { f.Type = content.CreatureTypeFey },
		"tags":      func(f *content.MonsterFields) { f.Tags = nil },
		"alignment": func(f *content.MonsterFields) { f.Alignment = content.AlignmentNeutral },
		"AC":        func(f *content.MonsterFields) { f.ArmorClass++ },
		"initiative": func(f *content.MonsterFields) {
			f.Initiative = must(content.NewStatBlockInitiative(2, 12))
		},
		"hit points": func(f *content.MonsterFields) {
			f.HitPoints = must(content.NewHitPointsEntry(10, dice.MustExpression(2, dice.D8(), 0)))
		},
		"speeds": func(f *content.MonsterFields) { f.Speeds = []content.SpeedEntry{monSpeed(t, 40)} },
		"abilities": func(f *content.MonsterFields) {
			f.Abilities = must(content.NewAbilityBlock(monScores(12), monSaves(1)))
		},
		"skills":          func(f *content.MonsterFields) { f.Skills = nil },
		"vulnerabilities": func(f *content.MonsterFields) { f.Vulnerabilities = f.Resistances },
		"resistances":     func(f *content.MonsterFields) { f.Resistances = nil },
		"immunities":      func(f *content.MonsterFields) { f.Immunities = f.Resistances },
		"gear":            func(f *content.MonsterFields) { f.Gear = nil },
		"senses":          func(f *content.MonsterFields) { f.Senses = nil },
		"passive":         func(f *content.MonsterFields) { f.PassivePerception++ },
		"languages":       func(f *content.MonsterFields) { f.Languages = content.LanguagesEntry{} },
		"CR":              func(f *content.MonsterFields) { f.ChallengeRating = content.MustChallengeRating("6") },
		"XP":              func(f *content.MonsterFields) { f.ExperiencePoints++ },
		"lair XP":         func(f *content.MonsterFields) { f.ExperiencePointsInLair++ },
		"traits":          func(f *content.MonsterFields) { f.Traits = nil },
		"actions":         func(f *content.MonsterFields) { f.Actions = nil },
		"bonus actions": func(f *content.MonsterFields) {
			f.BonusActions = []content.StatBlockPart{monPart(t, "Dash", content.StatBlockPartFields{})}
		},
		"reactions": func(f *content.MonsterFields) {
			f.Reactions = []content.StatBlockPart{monPart(t, "Parry", content.StatBlockPartFields{})}
		},
		"legendary": func(f *content.MonsterFields) { f.LegendaryActions = content.LegendaryActions{} },
	}
	for name, mutate := range mutators {
		f := rich(t)
		mutate(&f)
		if base.Equal(monMake(t, f)) {
			t.Errorf("changing %s left the monsters equal [FR-006]", name)
		}
	}
	// Immutability: the inputs are copied and every returned collection is a copy.
	f := rich(t)
	m := monMake(t, f)
	f.Sizes[0], f.Tags[0], f.Gear[0], f.Skills[content.SkillStealth] = content.SizeTiny, "Changed", "Changed", 99
	f.Actions[0] = monPart(t, "Changed", content.StatBlockPartFields{})
	if !m.Equal(base) {
		t.Error("changing a collection passed to the constructor changed the monster [FR-005]")
	}
	m.Sizes()[0], m.Tags()[0], m.Gear()[0] = content.SizeTiny, "Changed", "Changed"
	m.Skills()[content.SkillStealth] = 99
	m.Speeds()[0] = monSpeed(t, 99)
	m.Actions()[0] = monPart(t, "Changed", content.StatBlockPartFields{})
	m.Resistances()[0], m.Senses()[0] = nil, content.SenseEntry{}
	m.Traits()[0] = content.StatBlockPart{}
	if !m.Equal(base) {
		t.Error("changing a returned collection changed the monster [FR-005]")
	}
	if m.ProficiencyBonus() != 3 || m.AbilityModifier(ability.Strength).Value() != 0 || m.ProficientInSave(ability.Strength) {
		t.Error("derived accessors disagree with CR 5 and ten-for-all abilities [FR-019, FR-031]")
	}
	if xp, ok := m.ExperiencePointsInLair(); !ok || xp != 200 {
		t.Errorf("ExperiencePointsInLair = %d, %v [FR-032]", xp, ok)
	}
	if _, ok := base.Swarm(); ok {
		t.Error("a monster built without a swarm reports one [FR-009]")
	}
}

// Requirement: monSpec FR-008, FR-009, FR-011, FR-013, FR-016, FR-024, FR-025, FR-026, FR-032, FR-036, FR-099 -
// the rules between members of one monster, with the path and verdict the constructor reports.
// Path: per case.
func TestNewMonsterListRules(t *testing.T) {
	t.Parallel()
	dmg := func(d content.DamageType) content.DefenseEntry { return must(content.NewDamageTypeDefense(d, "")) }
	cond := func(c content.ConditionKind) content.DefenseEntry { return must(content.NewConditionDefense(c, "")) }
	described := must(content.NewDescribedDefense("Described"))
	cases := []struct {
		name, requirement, path string
		mutate                  func(*content.MonsterFields)
		wantOK                  bool
	}{
		{"no sizes", "FR-008", "unhappy (empty)", func(f *content.MonsterFields) { f.Sizes = nil }, false},
		{"two sizes", "FR-008", "happy", func(f *content.MonsterFields) {
			f.Sizes = []content.CreatureSize{content.SizeMedium, content.SizeSmall}
		}, true},
		{"duplicate sizes", "FR-008", "unhappy (invalid input)", func(f *content.MonsterFields) {
			f.Sizes = []content.CreatureSize{content.SizeMedium, content.SizeMedium}
		}, false},
		{"invalid size", "FR-008", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Sizes = []content.CreatureSize{0} }, false},
		{"swarm", "FR-009", "happy", func(f *content.MonsterFields) { f.Swarm = must(content.NewSwarmComposition(content.SizeTiny)) }, true},
		{"invalid type", "FR-050", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Type = 0 }, false},
		{"invalid alignment", "FR-050", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Alignment = 0 }, false},
		{"blank tag", "FR-011", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Tags = []string{" "} }, false},
		{"tags", "FR-011", "happy", func(f *content.MonsterFields) { f.Tags = []string{"Dinosaur", "Titan"} }, true},
		{"AC -1", "FR-013", "unhappy (boundary)", func(f *content.MonsterFields) { f.ArmorClass = -1 }, false},
		{"AC 0", "FR-013", "unhappy (boundary)", func(f *content.MonsterFields) { f.ArmorClass = 0 }, true},
		{"zero hit points", "FR-015", "unhappy (invalid input)", func(f *content.MonsterFields) { f.HitPoints = content.HitPointsEntry{} }, false},
		{"no speeds", "FR-016", "unhappy (empty)", func(f *content.MonsterFields) { f.Speeds = nil }, false},
		{"zero speed entry", "FR-016", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Speeds = []content.SpeedEntry{{}} }, false},
		{"duplicate speed", "FR-016", "unhappy (invalid input)", func(f *content.MonsterFields) {
			f.Speeds = []content.SpeedEntry{monSpeed(t, 30), monSpeed(t, 30)}
		}, false},
		{"same modes, other feet", "FR-016", "unhappy (invalid input)", func(f *content.MonsterFields) {
			f.Speeds = []content.SpeedEntry{monSpeed(t, 30), monSpeed(t, 40)}
		}, false},
		{"same modes, other qualifier", "FR-017", "happy", func(f *content.MonsterFields) {
			q := must(content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, 40, false, "wolf form only"))
			f.Speeds = []content.SpeedEntry{monSpeed(t, 30), q}
		}, true},
		{"zero abilities", "FR-018", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Abilities = content.AbilityBlock{} }, false},
		{"unknown skill", "FR-021", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Skills = map[content.Skill]int{0: 1} }, false},
		{"resistance to a condition", "FR-024", "unhappy (invalid input)", func(f *content.MonsterFields) {
			f.Resistances = []content.DefenseEntry{cond(content.ConditionPoisoned)}
		}, false},
		{"vulnerability to a condition", "FR-024", "unhappy (invalid input)", func(f *content.MonsterFields) {
			f.Vulnerabilities = []content.DefenseEntry{cond(content.ConditionPoisoned)}
		}, false},
		{"damage then conditions", "FR-025", "happy", func(f *content.MonsterFields) {
			f.Immunities = []content.DefenseEntry{dmg(content.DamagePoison), described, cond(content.ConditionExhaustion), cond(content.ConditionProne)}
		}, true},
		{"condition before damage", "FR-025", "unhappy (rule interaction)", func(f *content.MonsterFields) {
			f.Immunities = []content.DefenseEntry{cond(content.ConditionExhaustion), dmg(content.DamagePoison)}
		}, false},
		{"zero defence entry", "FR-024", "unhappy (invalid input)", func(f *content.MonsterFields) {
			f.Immunities = []content.DefenseEntry{content.DamageTypeDefense{}}
		}, false},
		{"blank gear", "FR-026", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Gear = []string{""} }, false},
		{"duplicate gear", "FR-026", "happy", func(f *content.MonsterFields) { f.Gear = []string{"Item", "Item"} }, true},
		{"zero sense entry", "FR-028", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Senses = []content.SenseEntry{{}} }, false},
		{"negative XP", "FR-032", "unhappy (boundary)", func(f *content.MonsterFields) { f.ExperiencePoints = -1 }, false},
		{"lair XP equal to XP", "FR-032", "unhappy (boundary)", func(f *content.MonsterFields) { f.ExperiencePoints, f.ExperiencePointsInLair = 10, 10 }, false},
		{"lair XP above XP", "FR-032", "unhappy (boundary)", func(f *content.MonsterFields) { f.ExperiencePoints, f.ExperiencePointsInLair = 10, 11 }, true},
		{"XP not checked against the CR row", "FR-032", "unhappy (boundary)", func(f *content.MonsterFields) {
			f.ChallengeRating, f.ExperiencePoints = content.MustChallengeRating("12"), 8000
		}, true},
		{"zero part", "FR-034", "unhappy (invalid input)", func(f *content.MonsterFields) { f.Traits = []content.StatBlockPart{{}} }, false},
		{"one Multiattack", "FR-036", "happy", func(f *content.MonsterFields) {
			f.Actions = []content.StatBlockPart{monPart(t, "Multiattack", content.StatBlockPartFields{})}
		}, true},
		{"second Multiattack", "FR-036", "unhappy (rule interaction)", func(f *content.MonsterFields) {
			p := monPart(t, "Multiattack", content.StatBlockPartFields{})
			f.Actions = []content.StatBlockPart{p, p}
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := monFields(t)
			c.mutate(&f)
			_, err := monBuild(t, f)
			if (err == nil) != c.wantOK || (err != nil && !isInvalidInput(err)) {
				t.Errorf("NewMonster error = %v, want ok=%v [%s, %s]", err, c.wantOK, c.requirement, c.path)
			}
		})
	}
}

// Requirement: monSpec FR-017, FR-089 - a monster accepts every value of each vocabulary type it uses.
// Path: happy.
func TestMonsterAcceptsEveryVocabularyValue(t *testing.T) {
	t.Parallel()
	for z := content.SizeTiny; z.Valid(); z++ {
		f := monFields(t)
		f.Sizes = []content.CreatureSize{z}
		monMake(t, f)
	}
	for ty := content.CreatureType(1); ty.Valid(); ty++ {
		f := monFields(t)
		f.Type = ty
		monMake(t, f)
	}
	for a := content.Alignment(1); a.Valid(); a++ {
		f := monFields(t)
		f.Alignment = a
		monMake(t, f)
	}
	for s := content.Skill(1); s.Valid(); s++ {
		f := monFields(t)
		f.Skills = map[content.Skill]int{s: 1}
		monMake(t, f)
	}
	for d := content.DamageAcid; d.Valid(); d++ {
		f := monFields(t)
		f.Immunities = []content.DefenseEntry{must(content.NewDamageTypeDefense(d, ""))}
		monMake(t, f)
	}
	for c := content.ConditionKind(1); c.Valid(); c++ {
		f := monFields(t)
		f.Immunities = []content.DefenseEntry{must(content.NewConditionDefense(c, ""))}
		monMake(t, f)
	}
	for s := content.SpecialSense(1); s.Valid(); s++ {
		f := monFields(t)
		f.Senses = []content.SenseEntry{must(content.NewSenseEntry(s, 10, ""))}
		monMake(t, f)
	}
}

// Requirement: monSpec FR-015 - Hit Points: average at least 1, count at least 1, die one of d4 d6 d8 d10 d12 d20,
// modifier any integer.
// Path: per case.
func TestHitPointsEntry(t *testing.T) {
	t.Parallel()
	d := func(count, sides, mod int) dice.Expression {
		return dice.MustExpression(count, dice.MustDie(sides), mod)
	}
	cases := []struct {
		name, path string
		avg        int
		e          dice.Expression
		wantOK     bool
	}{
		{"average 1", "unhappy (boundary)", 1, d(1, 4, 0), true},
		{"average 0", "unhappy (boundary)", 0, d(1, 4, 0), false},
		{"count 1", "unhappy (boundary)", 5, d(1, 8, 0), true},
		{"zero expression", "unhappy (boundary)", 5, dice.Expression{}, false},
		{"d3 is not allowed", "unhappy (boundary)", 5, d(1, 3, 0), false},
		{"d7 is not allowed", "unhappy (boundary)", 5, d(1, 7, 0), false},
		{"d100 is not allowed", "unhappy (boundary)", 5, d(1, 100, 0), false},
		{"d20 with a negative modifier", "happy", 5, d(2, 20, -4), true},
		{"average above the limit", "unhappy (boundary)", overLimit(), d(1, 4, 0), false},
	}
	for _, c := range cases {
		h, err := content.NewHitPointsEntry(c.avg, c.e)
		if (err == nil) != c.wantOK || (err != nil && !isInvalidInput(err)) {
			t.Errorf("%s: NewHitPointsEntry error = %v, want ok=%v [FR-015, %s]", c.name, err, c.wantOK, c.path)
		}
		if err == nil && (h.Average() != c.avg || h.Dice() != c.e) {
			t.Errorf("%s: accessors = %d, %v [FR-015]", c.name, h.Average(), h.Dice())
		}
	}
	for _, sides := range content.MonsterHitPointDieFaces() {
		if _, err := content.NewHitPointsEntry(3, d(1, sides, 0)); err != nil {
			t.Errorf("d%d rejected: %v [FR-015]", sides, err)
		}
	}
	if got := content.MonsterHitPointDieFaces(); !slices.Equal(got, []int{4, 6, 8, 10, 12, 20}) {
		t.Errorf("MonsterHitPointDieFaces = %v [FR-095]", got)
	}
}

// Requirement: monSpec FR-014 - Initiative holds the printed modifier and score, neither computed from the other.
// Path: happy.
func TestStatBlockInitiative(t *testing.T) {
	t.Parallel()
	i := must(content.NewStatBlockInitiative(-2, 13))
	if i.Modifier() != -2 || i.Score() != 13 {
		t.Errorf("Initiative = %d (%d), want -2 (13) as printed [FR-014]", i.Modifier(), i.Score())
	}
	if _, err := content.NewStatBlockInitiative(overLimit(), 0); !isInvalidInput(err) {
		t.Errorf("a modifier above the limit: error = %v [FR-014]", err)
	}
	if _, err := content.NewStatBlockInitiative(0, -content.MonsterMaxNumber-1); !isInvalidInput(err) {
		t.Errorf("a score below the limit: error = %v [FR-014]", err)
	}
}

// Requirement: monSpec FR-016, FR-017, table 4.3a - the speed rows.
// Path: per case.
func TestSpeedEntry(t *testing.T) {
	t.Parallel()
	modes := func(m ...content.SpeedMode) []content.SpeedMode { return m }
	cases := []struct {
		name, path string
		modes      []content.SpeedMode
		feet       int
		hover      bool
		qualifier  string
		wantOK     bool
	}{
		{"walk only", "happy", modes(content.SpeedWalk), 25, false, "", true},
		{"hover", "happy", modes(content.SpeedFly), 90, true, "", true},
		{"form qualifier", "happy", modes(content.SpeedWalk), 40, false, "wolf form only", true},
		{"alternative modes", "happy", modes(content.SpeedClimb, content.SpeedFly), 20, false, "GM’s choice", true},
		{"zero speed value", "unhappy (boundary)", modes(content.SpeedWalk), 0, false, "", true},
		{"negative speed", "unhappy (invalid input)", modes(content.SpeedWalk), -5, false, "", false},
		{"hover without fly", "unhappy (invalid input)", modes(content.SpeedSwim), 30, true, "", false},
		{"no modes", "unhappy (empty)", nil, 30, false, "", false},
		{"repeated mode", "unhappy (invalid input)", modes(content.SpeedWalk, content.SpeedWalk), 30, false, "", false},
		{"invalid mode", "unhappy (invalid input)", modes(0), 30, false, "", false},
		{"blank qualifier", "unhappy (invalid input)", modes(content.SpeedWalk), 30, false, " ", false},
		{"feet above the limit", "unhappy (boundary)", modes(content.SpeedWalk), overLimit(), false, "", false},
	}
	for _, c := range cases {
		e, err := content.NewSpeedEntry(c.modes, c.feet, c.hover, c.qualifier)
		if (err == nil) != c.wantOK || (err != nil && !isInvalidInput(err)) {
			t.Errorf("%s: NewSpeedEntry error = %v, want ok=%v [FR-016, %s]", c.name, err, c.wantOK, c.path)
			continue
		}
		if err == nil && (!slices.Equal(e.Modes(), c.modes) || e.Feet() != c.feet || e.Hover() != c.hover || e.Qualifier() != c.qualifier) {
			t.Errorf("%s: accessors disagree with the arguments [FR-017]", c.name)
		}
	}
	in := modes(content.SpeedWalk)
	e := must(content.NewSpeedEntry(in, 30, false, ""))
	in[0] = content.SpeedFly
	e.Modes()[0] = content.SpeedSwim
	if e.Modes()[0] != content.SpeedWalk {
		t.Error("a speed shares its modes with the caller [FR-005]")
	}
	for m, want := range map[string]content.SpeedMode{"walk": content.SpeedWalk, "burrow": content.SpeedBurrow, "climb": content.SpeedClimb, "fly": content.SpeedFly, "swim": content.SpeedSwim} {
		got, err := content.ParseSpeedMode(m)
		if err != nil || got != want || got.String() != m {
			t.Errorf("ParseSpeedMode(%q) = %v, %v [FR-016]", m, got, err)
		}
	}
	if _, err := content.ParseSpeedMode("Walk"); !isInvalidInput(err) {
		t.Error("ParseSpeedMode accepted a different case [FR-016]")
	}
}

// Requirement: monSpec FR-018, FR-019, FR-020 - the ability block composes the kernel scores with one printed save
// each; the modifier is delegated, proficiency is a save that differs from the modifier.
// Path: per case.
func TestAbilityBlock(t *testing.T) {
	t.Parallel()
	scores := must(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(21), ability.Dexterity: ability.MustScore(10), ability.Constitution: ability.MustScore(10),
		ability.Intelligence: ability.MustScore(18), ability.Wisdom: ability.MustScore(10), ability.Charisma: ability.MustScore(1),
	}))
	saves := monSaves(0)
	saves[ability.Strength], saves[ability.Intelligence], saves[ability.Charisma] = 5, 8, -5
	b := must(content.NewAbilityBlock(scores, saves))
	if b.AbilityModifier(ability.Strength) != scores.Modifier(ability.Strength) || b.Scores() != scores {
		t.Error("the ability modifier is not the kernel's [FR-019]")
	}
	for a, want := range map[ability.Ability]bool{ability.Strength: false, ability.Intelligence: true, ability.Charisma: false, ability.Wisdom: false} {
		if got := b.ProficientInSave(a); got != want {
			t.Errorf("ProficientInSave(%v) = %v, want %v [FR-020]", a, got, want)
		}
	}
	if b.Save(ability.Intelligence) != 8 || b.Save(0) != 0 || b.ProficientInSave(0) || b.IsZero() || !(content.AbilityBlock{}).IsZero() {
		t.Error("Save, ProficientInSave or IsZero disagree [FR-018]")
	}
	missing := monSaves(0)
	delete(missing, ability.Wisdom)
	badKey := monSaves(0)
	badKey[0] = 1
	bigSave := monSaves(0)
	bigSave[ability.Wisdom] = overLimit()
	for name, c := range map[string]struct {
		s     ability.Scores
		saves map[ability.Ability]int
	}{
		"zero scores": {ability.Scores{}, monSaves(0)}, "missing save": {scores, missing},
		"invalid key": {scores, badKey}, "save out of range": {scores, bigSave}, "no saves": {scores, nil},
	} {
		if _, err := content.NewAbilityBlock(c.s, c.saves); !isInvalidInput(err) {
			t.Errorf("%s: NewAbilityBlock error = %v, want ErrInvalidInput [FR-018]", name, err)
		}
	}
}

// Requirement: monSpec FR-024, table 4.4a - the three defence variants and their qualifiers.
// Path: per case.
func TestDefenseEntries(t *testing.T) {
	t.Parallel()
	d := must(content.NewDamageTypeDefense(content.DamagePiercing, "from weapons"))
	if d.DamageType() != content.DamagePiercing || d.Qualifier() != "from weapons" {
		t.Error("damage defence accessors [FR-024]")
	}
	c := must(content.NewConditionDefense(content.ConditionCharmed, "with Mind Blank"))
	if c.Condition() != content.ConditionCharmed || c.Qualifier() != "with Mind Blank" {
		t.Error("condition defence accessors [FR-024]")
	}
	if must(content.NewDescribedDefense("Chosen type")).Text() != "Chosen type" {
		t.Error("described defence accessor [FR-024]")
	}
	if _, err := content.NewDamageTypeDefense(0, ""); !isInvalidInput(err) {
		t.Error("an invalid damage type was accepted [FR-024]")
	}
	if _, err := content.NewDamageTypeDefense(content.DamageFire, " "); !isInvalidInput(err) {
		t.Error("a blank qualifier was accepted [FR-024]")
	}
	if _, err := content.NewConditionDefense(0, ""); !isInvalidInput(err) {
		t.Error("an invalid condition was accepted [FR-024]")
	}
	if _, err := content.NewConditionDefense(content.ConditionProne, "\n"); !isInvalidInput(err) {
		t.Error("a blank condition qualifier was accepted [FR-024]")
	}
	if _, err := content.NewDescribedDefense(""); !isInvalidInput(err) {
		t.Error("blank described text was accepted [FR-024]")
	}
}

// Requirement: monSpec FR-027, FR-028 - senses: kind, range at least 0, optional qualifier.
// Path: per case.
func TestSenseEntry(t *testing.T) {
	t.Parallel()
	e := must(content.NewSenseEntry(content.SenseDarkvision, 0, "unimpeded by magical Darkness"))
	if e.Sense() != content.SenseDarkvision || e.Feet() != 0 || e.Qualifier() != "unimpeded by magical Darkness" {
		t.Error("sense accessors [FR-027]")
	}
	for name, args := range map[string]struct {
		s    content.SpecialSense
		feet int
		q    string
	}{
		"unknown kind": {0, 10, ""}, "negative range": {content.SenseBlindsight, -1, ""},
		"range above the limit": {content.SenseBlindsight, overLimit(), ""}, "blank qualifier": {content.SenseTruesight, 10, " "},
	} {
		if _, err := content.NewSenseEntry(args.s, args.feet, args.q); !isInvalidInput(err) {
			t.Errorf("%s: NewSenseEntry error = %v, want ErrInvalidInput [FR-028]", name, err)
		}
	}
}

// Requirement: monSpec FR-029, table 4.5b - languages: phrases verbatim, optional telepathy and qualifier.
// Path: per case.
func TestLanguagesEntry(t *testing.T) {
	t.Parallel()
	none := must(content.NewLanguagesEntry(nil))
	if _, ok := none.Telepathy(); ok || len(none.Phrases()) != 0 || !none.Equal(content.LanguagesEntry{}) {
		t.Error("None is not the zero entry [FR-029]")
	}
	l := must(must(content.NewLanguagesEntry([]string{"Deep Speech", "Understands A but can’t speak"})).WithTelepathy(120, "doesn’t allow a reply"))
	if feet, ok := l.Telepathy(); !ok || feet != 120 || l.TelepathyQualifier() != "doesn’t allow a reply" || len(l.Phrases()) != 2 {
		t.Error("languages accessors [FR-029]")
	}
	if l.Equal(none) || none.Equal(l) || l.Equal(must(l.WithTelepathy(60, "doesn’t allow a reply"))) {
		t.Error("languages Equal ignores telepathy [FR-029]")
	}
	in := []string{"Common"}
	e := must(content.NewLanguagesEntry(in))
	in[0] = "Changed"
	e.Phrases()[0] = "Changed"
	if e.Phrases()[0] != "Common" {
		t.Error("languages share their phrases [FR-005]")
	}
	if _, err := content.NewLanguagesEntry([]string{"Common", ""}); !isInvalidInput(err) {
		t.Error("a blank phrase was accepted [FR-029]")
	}
	if _, err := none.WithTelepathy(-1, ""); !isInvalidInput(err) {
		t.Error("negative telepathy was accepted [FR-029]")
	}
	if _, err := none.WithTelepathy(10, " "); !isInvalidInput(err) {
		t.Error("a blank telepathy qualifier was accepted [FR-029]")
	}
}

// Requirement: monSpec FR-035, table 4.6a - usage variants and their bounds.
// Path: per case.
func TestPartUsage(t *testing.T) {
	t.Parallel()
	if p := must(content.NewPartPerDay(3, 4)); p.Count() != 3 {
		t.Error("PartPerDay count [FR-035]")
	} else if n, ok := p.InLairCount(); !ok || n != 4 {
		t.Error("PartPerDay lair count [FR-035]")
	}
	if _, ok := must(content.NewPartPerDay(3, 0)).InLairCount(); ok {
		t.Error("PartPerDay reports a lair count that was not printed [FR-035]")
	}
	for name, args := range map[string][2]int{"zero uses": {0, 0}, "negative lair": {1, -1}, "above the limit": {overLimit(), 0}} {
		if _, err := content.NewPartPerDay(args[0], args[1]); !isInvalidInput(err) {
			t.Errorf("%s: NewPartPerDay error = %v [FR-035]", name, err)
		}
	}
	for name, args := range map[string][2]int{"4-6": {4, 6}, "6-6": {6, 6}, "1-1": {1, 1}} {
		r, err := content.NewPartRecharge(args[0], args[1])
		if err != nil || r.Low() != args[0] || r.High() != args[1] {
			t.Errorf("%s: NewPartRecharge = %v, %v [FR-035]", name, r, err)
		}
	}
	for name, args := range map[string][2]int{"5-4": {5, 4}, "7": {7, 7}, "0-6": {0, 6}, "1-7": {1, 7}} {
		if _, err := content.NewPartRecharge(args[0], args[1]); !isInvalidInput(err) {
			t.Errorf("%s: NewPartRecharge error = %v [FR-035]", name, err)
		}
	}
	if must(content.NewPartUsageOther("1/Day; Requires Soul Bag")).Text() != "1/Day; Requires Soul Bag" {
		t.Error("PartUsageOther text [FR-035]")
	}
	if _, err := content.NewPartUsageOther(" "); !isInvalidInput(err) {
		t.Error("blank other usage was accepted [FR-035]")
	}
}

// Requirement: monSpec FR-033, FR-034, FR-036, FR-038, FR-041 - a part has a name, a usage and text; clause labels
// are unique; a plan is only valid on the Multiattack.
// Path: per case.
func TestStatBlockPart(t *testing.T) {
	t.Parallel()
	clause := func(l content.EffectClauseLabel) content.EffectClause {
		return must(content.NewEffectClause(l, "Text.", nil, nil))
	}
	plan := must(content.NewMultiattackPlan([]content.MultiattackVariant{
		must(content.NewMultiattackVariant("", []content.AttackSlotGroup{must(content.NewAttackSlotGroup(2, []string{"Bite"}))}, nil, nil)),
	}))
	cases := []struct {
		name, part string
		f          content.StatBlockPartFields
		wantOK     bool
	}{
		{"valid", "Bite", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T."}, true},
		{"blank name", " ", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T."}, false},
		{"nil usage", "Bite", content.StatBlockPartFields{Text: "T."}, false},
		{"zero usage variant", "Bite", content.StatBlockPartFields{Usage: content.PartPerDay{}, Text: "T."}, false},
		{"blank text", "Bite", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: " "}, false},
		{"several failure labels", "Bite", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Clauses: []content.EffectClause{
			clause(content.ClauseFirstFailure), clause(content.ClauseSecondFailure), clause(content.ClauseFailure),
		}}, true},
		{"label twice", "Bite", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Clauses: []content.EffectClause{
			clause(content.ClauseHit), clause(content.ClauseHit),
		}}, false},
		{"zero clause", "Bite", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Clauses: []content.EffectClause{{}}}, false},
		{"plan on Multiattack", "Multiattack", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Plan: plan}, true},
		{"plan on another part", "Bite", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Plan: plan}, false},
	}
	for _, c := range cases {
		p, err := content.NewStatBlockPart(c.part, c.f)
		if (err == nil) != c.wantOK || (err != nil && !isInvalidInput(err)) {
			t.Errorf("%s: NewStatBlockPart error = %v, want ok=%v [FR-034]", c.name, err, c.wantOK)
			continue
		}
		if err == nil && (p.Name() != c.part || p.Text() != c.f.Text || p.Usage() != c.f.Usage || p.IsZero()) {
			t.Errorf("%s: accessors [FR-034]", c.name)
		}
	}
	p := monPart(t, "Bite", content.StatBlockPartFields{})
	if _, ok := p.Attack(); ok {
		t.Error("a part reports an attack it was not given [FR-037]")
	}
	if _, ok := p.SavingThrow(); ok {
		t.Error("a part reports a saving throw it was not given [FR-039]")
	}
	if _, ok := p.Spellcasting(); ok {
		t.Error("a part reports spellcasting it was not given [FR-044]")
	}
	if _, ok := p.Plan(); ok {
		t.Error("a part reports a plan it was not given [FR-036]")
	}
	if !(content.StatBlockPart{}).IsZero() {
		t.Error("the zero part is not zero [FR-034]")
	}
}

// Requirement: monSpec FR-037, FR-039, FR-038, FR-100, FR-042, FR-043, FR-101, tables 4.6b, 4.7a - attack, save,
// clause and damage notation and their bounds.
// Path: per case.
func TestNotation(t *testing.T) {
	t.Parallel()
	rng := must(content.NewStatBlockAttackRange(80, 320))
	if long, ok := rng.Long(); !ok || long != 320 || rng.Normal() != 80 {
		t.Error("range accessors [FR-037]")
	}
	if _, ok := must(content.NewStatBlockAttackRange(60, 0)).Long(); ok {
		t.Error("a single range reports a long range [FR-037]")
	}
	for name, args := range map[string][2]int{"normal 0": {0, 0}, "long equal": {60, 60}, "long below": {60, 30}, "negative long": {60, -1}} {
		if _, err := content.NewStatBlockAttackRange(args[0], args[1]); !isInvalidInput(err) {
			t.Errorf("%s: NewStatBlockAttackRange error = %v [FR-037]", name, err)
		}
	}
	a := must(content.NewStatBlockAttack(content.StatBlockAttackFields{
		Kind: content.AttackMeleeOrRanged, Bonus: -1, Qualifier: "with Advantage", Reach: 10, HasReach: true, Range: rng,
	}))
	if reach, ok := a.Reach(); !ok || reach != 10 || a.Bonus() != -1 || a.Kind() != content.AttackMeleeOrRanged || a.Qualifier() != "with Advantage" {
		t.Error("attack accessors [FR-037]")
	}
	if r, ok := a.Range(); !ok || r != rng {
		t.Error("attack range accessor [FR-037]")
	}
	plain := must(content.NewStatBlockAttack(content.StatBlockAttackFields{Kind: content.AttackMelee, Bonus: 4, Reach: 7}))
	if _, ok := plain.Reach(); ok || plain.Qualifier() != "" {
		t.Error("an attack with no reach reports one [FR-037]")
	}
	if _, ok := plain.Range(); ok || plain.IsZero() || !(content.StatBlockAttack{}).IsZero() {
		t.Error("attack range or IsZero [FR-037]")
	}
	for name, f := range map[string]content.StatBlockAttackFields{
		"no kind": {Bonus: 1}, "bonus above the limit": {Kind: content.AttackMelee, Bonus: overLimit()},
		"negative reach": {Kind: content.AttackMelee, Reach: -1, HasReach: true}, "blank qualifier": {Kind: content.AttackMelee, Qualifier: " "},
		"bad range": {Kind: content.AttackRanged, Range: content.StatBlockAttackRange{}},
	} {
		_, err := content.NewStatBlockAttack(f)
		if name == "bad range" {
			if err != nil {
				t.Errorf("the zero range means none: error = %v [FR-037]", err)
			}
			continue
		}
		if !isInvalidInput(err) {
			t.Errorf("%s: NewStatBlockAttack error = %v [FR-037]", name, err)
		}
	}
	s := must(content.NewStatBlockSavingThrow(ability.Wisdom, 15, "the triggering creature", "Response—"))
	if s.Ability() != ability.Wisdom || s.DC() != 15 || s.Target() != "the triggering creature" || s.Prefix() != "Response—" || s.IsZero() {
		t.Error("saving throw accessors [FR-039]")
	}
	for name, args := range map[string]struct {
		a      ability.Ability
		dc     int
		t, pre string
	}{"DC 0": {ability.Wisdom, 0, "", ""}, "no ability": {0, 12, "", ""}, "blank target": {ability.Wisdom, 12, " ", ""}, "blank prefix": {ability.Wisdom, 12, "", "\t"}} {
		if _, err := content.NewStatBlockSavingThrow(args.a, args.dc, args.t, args.pre); !isInvalidInput(err) {
			t.Errorf("%s: NewStatBlockSavingThrow error = %v [FR-039]", name, err)
		}
	}
	if !(content.StatBlockSavingThrow{}).IsZero() {
		t.Error("the zero saving throw is not zero [FR-039]")
	}
	// Damage.
	fire := must(content.NewSpecificDamageType(content.DamageFire))
	cold := must(content.NewSpecificDamageType(content.DamageCold))
	described := must(content.NewDescribedDamageType("damage of the chosen type"))
	d1d6 := dice.MustExpression(1, dice.D6(), 2)
	for name, c := range map[string]struct {
		number int
		e      dice.Expression
		ref    content.DamageTypeRef
		flag   bool
		wantOK bool
	}{
		"number and dice": {5, d1d6, fire, false, true}, "fixed only": {1, dice.Expression{}, cold, false, true},
		"described type": {7, d1d6, described, false, true}, "flag on Fire": {5, d1d6, fire, true, true},
		"flag on Cold": {5, d1d6, cold, true, false}, "flag on a described type": {5, d1d6, described, true, false},
		"negative number": {-1, d1d6, fire, false, false}, "number above the limit": {overLimit(), d1d6, fire, false, false},
		"d7": {4, dice.MustExpression(1, dice.MustDie(7), 0), fire, false, false}, "d100": {4, dice.MustExpression(1, dice.D100(), 0), fire, false, false},
		"nil type": {4, d1d6, nil, false, false}, "zero type": {4, d1d6, content.SpecificDamageType{}, false, false},
		"zero described type": {4, d1d6, content.DescribedDamageType{}, false, false}, "zero number and no dice": {0, dice.Expression{}, fire, false, true},
	} {
		dm, err := content.NewStatBlockDamage(c.number, c.e, c.ref, c.flag)
		if (err == nil) != c.wantOK || (err != nil && !isInvalidInput(err)) {
			t.Errorf("%s: NewStatBlockDamage error = %v, want ok=%v [FR-042, FR-043, FR-101]", name, err, c.wantOK)
			continue
		}
		if err == nil {
			_, hasDice := dm.Dice()
			if dm.Number() != c.number || hasDice != !c.e.IsZero() || dm.Type() != c.ref || dm.UnderwaterResistanceDenied() != c.flag || dm.IsZero() {
				t.Errorf("%s: damage accessors [FR-042]", name)
			}
		}
	}
	if _, err := content.NewSpecificDamageType(0); !isInvalidInput(err) {
		t.Error("an invalid damage type was accepted [FR-043]")
	}
	if _, err := content.NewDescribedDamageType(" "); !isInvalidInput(err) {
		t.Error("blank described damage was accepted [FR-042]")
	}
	// Clauses and riders.
	hours := must(content.NewImmuneToPartForHours(24))
	for name, c := range map[string]struct {
		label  content.EffectClauseLabel
		text   string
		rider  content.SaveSuccessRider
		wantOK bool
	}{
		"hit": {content.ClauseHit, "Text.", nil, true}, "success with half damage only": {content.ClauseSuccess, "Half damage only.", content.HalfDamageOnly{}, true},
		"success with half damage": {content.ClauseSuccess, "Half damage.", content.HalfDamage{}, true}, "success with immunity": {content.ClauseSuccess, "Immune.", hours, true},
		"rider on a failure": {content.ClauseFailure, "Text.", content.HalfDamage{}, false}, "rider on failure or success": {content.ClauseFailureOrSuccess, "T.", hours, false},
		"zero immunity": {content.ClauseSuccess, "T.", content.ImmuneToPartForHours{}, false}, "no label": {0, "T.", nil, false},
		"blank text": {content.ClauseHit, " ", nil, false},
	} {
		_, err := content.NewEffectClause(c.label, c.text, c.rider, nil)
		if (err == nil) != c.wantOK || (err != nil && !isInvalidInput(err)) {
			t.Errorf("%s: NewEffectClause error = %v, want ok=%v [FR-038, FR-100]", name, err, c.wantOK)
		}
	}
	if hours.Hours() != 24 {
		t.Error("rider hours [FR-100]")
	}
	for _, h := range []int{0, -1, overLimit()} {
		if _, err := content.NewImmuneToPartForHours(h); !isInvalidInput(err) {
			t.Errorf("NewImmuneToPartForHours(%d) error = %v [FR-100]", h, err)
		}
	}
	dm := must(content.NewStatBlockDamage(5, d1d6, fire, true))
	c := must(content.NewEffectClause(content.ClauseSuccess, "T.", hours, []content.StatBlockDamage{dm}))
	if r, ok := c.SuccessRider(); !ok || r != hours || c.Label() != content.ClauseSuccess || c.Text() != "T." || len(c.Damage()) != 1 {
		t.Error("clause accessors [FR-038]")
	}
	if _, err := content.NewEffectClause(content.ClauseHit, "T.", nil, []content.StatBlockDamage{{}}); !isInvalidInput(err) {
		t.Error("a zero damage instance was accepted [FR-042]")
	}
	if c.Equal(must(content.NewEffectClause(content.ClauseSuccess, "T.", hours, nil))) || !c.Equal(c) {
		t.Error("clause Equal ignores the damage [FR-038]")
	}
	for s, want := range map[string]content.EffectClauseLabel{
		"hit": content.ClauseHit, "miss": content.ClauseMiss, "hitOrMiss": content.ClauseHitOrMiss, "failure": content.ClauseFailure,
		"success": content.ClauseSuccess, "failureOrSuccess": content.ClauseFailureOrSuccess, "firstFailure": content.ClauseFirstFailure,
		"secondFailure": content.ClauseSecondFailure, "subsequentFailures": content.ClauseSubsequentFailures,
		"failureByFiveOrMore": content.ClauseFailureByFiveOrMore,
	} {
		if got, err := content.ParseEffectClauseLabel(s); err != nil || got != want || got.String() != s {
			t.Errorf("ParseEffectClauseLabel(%q) = %v, %v [FR-038]", s, got, err)
		}
	}
	for s, want := range map[string]content.StatBlockAttackKind{"melee": content.AttackMelee, "ranged": content.AttackRanged, "meleeOrRanged": content.AttackMeleeOrRanged} {
		if got, err := content.ParseStatBlockAttackKind(s); err != nil || got != want || got.String() != s {
			t.Errorf("ParseStatBlockAttackKind(%q) = %v, %v [FR-037]", s, got, err)
		}
	}
	if _, err := content.ParseEffectClauseLabel("Hit"); !isInvalidInput(err) {
		t.Error("ParseEffectClauseLabel accepted a different case [FR-038]")
	}
	if _, err := content.ParseStatBlockAttackKind("Melee"); !isInvalidInput(err) {
		t.Error("ParseStatBlockAttackKind accepted a different case [FR-037]")
	}
}

// Requirement: monSpec FR-040 - legendary actions: uses at least 1, in-lair at least 1 when present, actions
// not empty.
// Path: per case.
func TestLegendaryActions(t *testing.T) {
	t.Parallel()
	act := monPart(t, "Move", content.StatBlockPartFields{})
	l := must(content.NewLegendaryActions(3, 4, []content.StatBlockPart{act}))
	if inLair, ok := l.UsesInLair(); l.Uses() != 3 || !ok || inLair != 4 || len(l.Actions()) != 1 || l.IsZero() {
		t.Error("legendary actions accessors [FR-040]")
	}
	if _, ok := must(content.NewLegendaryActions(3, 0, []content.StatBlockPart{act})).UsesInLair(); ok {
		t.Error("a missing lair count is reported [FR-040]")
	}
	for name, c := range map[string]struct {
		uses, lair int
		actions    []content.StatBlockPart
	}{
		"zero uses": {0, 0, []content.StatBlockPart{act}}, "negative lair": {3, -1, []content.StatBlockPart{act}},
		"no actions": {3, 0, nil}, "zero action": {3, 0, []content.StatBlockPart{{}}},
	} {
		if _, err := content.NewLegendaryActions(c.uses, c.lair, c.actions); !isInvalidInput(err) {
			t.Errorf("%s: NewLegendaryActions error = %v [FR-040]", name, err)
		}
	}
	if !(content.LegendaryActions{}).IsZero() || l.Equal(must(content.NewLegendaryActions(3, 0, []content.StatBlockPart{act}))) {
		t.Error("legendary actions IsZero or Equal [FR-040]")
	}
}

// Requirement: monSpec FR-044, FR-045, FR-046, FR-047 - spellcasting: ability, DC, bonus, components, groups,
// frequencies and entries.
// Path: per case.
func TestMonsterSpellcasting(t *testing.T) {
	t.Parallel()
	entry := must(content.NewMonsterSpellEntry("Shapechange", "Beast form only, no Concentration", 0))
	leveled := must(content.NewMonsterSpellEntry("Detect Magic", "", 9))
	if lvl, ok := leveled.CastAtLevel(); !ok || lvl != 9 || leveled.Name() != "Detect Magic" || leveled.Note() != "" {
		t.Error("spell entry accessors [FR-046]")
	}
	if _, ok := entry.CastAtLevel(); ok {
		t.Error("an entry with no level reports one [FR-046]")
	}
	for _, lvl := range []int{-1, 10} {
		if _, err := content.NewMonsterSpellEntry("Spell", "", lvl); !isInvalidInput(err) {
			t.Errorf("castAtLevel %d error = %v [FR-046]", lvl, err)
		}
	}
	if _, err := content.NewMonsterSpellEntry("", "", 0); !isInvalidInput(err) {
		t.Error("a blank spell name was accepted [FR-046]")
	}
	if _, err := content.NewMonsterSpellEntry("Spell", " ", 0); !isInvalidInput(err) {
		t.Error("a blank note was accepted [FR-046]")
	}
	perDay := must(content.NewSpellFrequencyPerDay(3))
	each := must(content.NewSpellFrequencyPerDayEach(2))
	if perDay.Count() != 3 || each.Count() != 2 {
		t.Error("frequency counts [FR-045]")
	}
	for _, n := range []int{0, -1, overLimit()} {
		if _, err := content.NewSpellFrequencyPerDay(n); !isInvalidInput(err) {
			t.Errorf("PerDay(%d) error = %v [FR-045]", n, err)
		}
		if _, err := content.NewSpellFrequencyPerDayEach(n); !isInvalidInput(err) {
			t.Errorf("PerDayEach(%d) error = %v [FR-045]", n, err)
		}
	}
	for name, freq := range map[string]content.SpellFrequency{"at will": content.SpellFrequencyAtWill{}, "per day": perDay, "per day each": each} {
		if _, err := content.NewMonsterSpellGroup(freq, []content.MonsterSpellEntry{entry}); err != nil {
			t.Errorf("%s: NewMonsterSpellGroup error = %v [FR-045]", name, err)
		}
	}
	for name, c := range map[string]struct {
		f      content.SpellFrequency
		spells []content.MonsterSpellEntry
	}{
		"no spells": {perDay, nil}, "nil frequency": {nil, []content.MonsterSpellEntry{entry}},
		"zero frequency": {content.SpellFrequencyPerDay{}, []content.MonsterSpellEntry{entry}}, "zero spell": {perDay, []content.MonsterSpellEntry{{}}},
	} {
		if _, err := content.NewMonsterSpellGroup(c.f, c.spells); !isInvalidInput(err) {
			t.Errorf("%s: NewMonsterSpellGroup error = %v [FR-045]", name, err)
		}
	}
	group := must(content.NewMonsterSpellGroup(perDay, []content.MonsterSpellEntry{entry, leveled}))
	if group.Frequency() != perDay || len(group.Spells()) != 2 || group.Equal(must(content.NewMonsterSpellGroup(each, group.Spells()))) {
		t.Error("spell group accessors or Equal [FR-045]")
	}
	f := content.MonsterSpellcastingFields{
		Ability: ability.Intelligence, SaveDC: 17, HasSaveDC: true, AttackBonus: 9, HasAttackBonus: true,
		Components: "requiring no Material components", Groups: []content.MonsterSpellGroup{group},
	}
	s := must(content.NewMonsterSpellcasting(f))
	dc, hasDC := s.SaveDC()
	bonus, hasBonus := s.AttackBonus()
	if s.Ability() != ability.Intelligence || !hasDC || dc != 17 || !hasBonus || bonus != 9 || s.Components() != f.Components || len(s.Groups()) != 1 || s.IsZero() {
		t.Error("spellcasting accessors [FR-044]")
	}
	f.HasSaveDC, f.HasAttackBonus = false, false
	bare := must(content.NewMonsterSpellcasting(f))
	if _, ok := bare.SaveDC(); ok || s.Equal(bare) || !s.Equal(s) {
		t.Error("spellcasting without a DC or bonus [FR-044]")
	}
	f.Groups[0] = leveledGroup(t, leveled)
	if len(bare.Groups()) != 1 || bare.Groups()[0].Equal(f.Groups[0]) {
		t.Error("spellcasting shares its groups with the caller [FR-005]")
	}
	for name, mutate := range map[string]func(*content.MonsterSpellcastingFields){
		"no ability": func(f *content.MonsterSpellcastingFields) { f.Ability = 0 }, "no groups": func(f *content.MonsterSpellcastingFields) { f.Groups = nil },
		"zero group": func(f *content.MonsterSpellcastingFields) { f.Groups = []content.MonsterSpellGroup{{}} },
		"DC 0":       func(f *content.MonsterSpellcastingFields) { f.SaveDC, f.HasSaveDC = 0, true },
		"bonus above the limit": func(f *content.MonsterSpellcastingFields) {
			f.AttackBonus, f.HasAttackBonus = overLimit(), true
		},
		"blank components": func(f *content.MonsterSpellcastingFields) { f.Components = " " },
	} {
		g := content.MonsterSpellcastingFields{Ability: ability.Wisdom, Groups: []content.MonsterSpellGroup{group}}
		mutate(&g)
		if _, err := content.NewMonsterSpellcasting(g); !isInvalidInput(err) {
			t.Errorf("%s: NewMonsterSpellcasting error = %v [FR-044]", name, err)
		}
	}
	if !(content.MonsterSpellcasting{}).IsZero() {
		t.Error("the zero spellcasting is not zero [FR-044]")
	}
}

func leveledGroup(tb testing.TB, e content.MonsterSpellEntry) content.MonsterSpellGroup {
	tb.Helper()
	return must(content.NewMonsterSpellGroup(content.SpellFrequencyAtWill{}, []content.MonsterSpellEntry{e}))
}

// Requirement: monSpec FR-098, FR-099, table 4.6c - the Multiattack plan invariants and name resolution.
// Path: per case.
func TestMultiattackPlan(t *testing.T) {
	t.Parallel()
	group := func(count int, parts ...string) content.AttackSlotGroup {
		return must(content.NewAttackSlotGroup(count, parts))
	}
	variant := func(requires string, groups []content.AttackSlotGroup, uses []content.AdditionalUse, repl []content.ReplacementRule) content.MultiattackVariant {
		return must(content.NewMultiattackVariant(requires, groups, uses, repl))
	}
	opt := func(part string) content.ReplacementOption { return must(content.NewReplacementOption(part, "", 0)) }
	scope := func(part string) content.ReplacementScope {
		return must(content.NewReplacementScopeOneAttackPart(part))
	}
	// Group, replacement and conditional variants.
	g1 := group(3, "Foreleg", "Poison Burst")
	if g1.Count() != 3 || !slices.Equal(g1.Parts(), []string{"Foreleg", "Poison Burst"}) {
		t.Error("group accessors [FR-098]")
	}
	use := must(content.NewAdditionalUse([]string{"Visage"}))
	if !slices.Equal(use.Options(), []string{"Visage"}) || !use.Equal(use) || use.Equal(must(content.NewAdditionalUse([]string{"Other"}))) {
		t.Error("additional use accessors [FR-098]")
	}
	withSpell := must(content.NewReplacementOption("Spellcasting", "Scorching Ray", 3))
	if name, lvl, ok := withSpell.Spell(); !ok || name != "Scorching Ray" || lvl != 3 || withSpell.Part() != "Spellcasting" {
		t.Error("replacement option accessors [FR-098]")
	}
	if _, _, ok := opt("Breath").Spell(); ok {
		t.Error("an option with no spell reports one [FR-098]")
	}
	rule := must(content.NewReplacementRule(1, scope("Claw"), []content.ReplacementOption{opt("Breath")}, content.ReplaceOnce))
	if rule.Count() != 1 || rule.Scope() != scope("Claw") || len(rule.Options()) != 1 || rule.Limit() != content.ReplaceOnce {
		t.Error("replacement rule accessors [FR-098]")
	}
	chimera := variant("", []content.AttackSlotGroup{group(1, "Ram"), group(1, "Bite"), group(1, "Claw")}, nil, []content.ReplacementRule{rule})
	hasten := variant("Hasten", []content.AttackSlotGroup{group(3, "Slam")}, nil, nil)
	base := variant("", []content.AttackSlotGroup{group(2, "Slam")}, []content.AdditionalUse{use}, nil)
	if r, ok := hasten.RequiresUsed(); !ok || r != "Hasten" || len(base.Groups()) != 1 || len(base.AdditionalUses()) != 1 || len(chimera.Replacements()) != 1 {
		t.Error("variant accessors [FR-098]")
	}
	if _, ok := base.RequiresUsed(); ok {
		t.Error("a variant with no requirement reports one [FR-098]")
	}
	plan := must(content.NewMultiattackPlan([]content.MultiattackVariant{base, hasten}))
	if len(plan.Variants()) != 2 || plan.IsZero() || !plan.Equal(plan) || plan.Equal(must(content.NewMultiattackPlan([]content.MultiattackVariant{base}))) {
		t.Error("plan accessors or Equal [FR-098]")
	}
	if !base.Equal(base) || base.Equal(hasten) || !rule.Equal(rule) || !group(1, "A").Equal(group(1, "A")) || group(1, "A").Equal(group(2, "A")) {
		t.Error("Equal of plan members [FR-098]")
	}
	// Invalid arguments.
	anyScope := content.ReplacementScopeAnyAttack{}
	badCases := map[string]func() error{
		"group count 0":          func() error { _, err := content.NewAttackSlotGroup(0, []string{"A"}); return err },
		"group no parts":         func() error { _, err := content.NewAttackSlotGroup(1, nil); return err },
		"group repeated part":    func() error { _, err := content.NewAttackSlotGroup(1, []string{"A", "A"}); return err },
		"group blank part":       func() error { _, err := content.NewAttackSlotGroup(1, []string{" "}); return err },
		"additional none":        func() error { _, err := content.NewAdditionalUse(nil); return err },
		"additional repeat":      func() error { _, err := content.NewAdditionalUse([]string{"A", "A"}); return err },
		"option level 10":        func() error { _, err := content.NewReplacementOption("P", "S", 10); return err },
		"option level, no spell": func() error { _, err := content.NewReplacementOption("P", "", 3); return err },
		"option blank part":      func() error { _, err := content.NewReplacementOption("", "", 0); return err },
		"option blank spell":     func() error { _, err := content.NewReplacementOption("P", " ", 0); return err },
		"scope blank part":       func() error { _, err := content.NewReplacementScopeOneAttackPart(""); return err },
		"rule count 0": func() error {
			_, err := content.NewReplacementRule(0, anyScope, []content.ReplacementOption{opt("A")}, content.ReplaceOnce)
			return err
		},
		"rule nil scope": func() error {
			_, err := content.NewReplacementRule(1, nil, []content.ReplacementOption{opt("A")}, content.ReplaceOnce)
			return err
		},
		"rule zero scope": func() error {
			_, err := content.NewReplacementRule(1, content.ReplacementScopeOneAttackPart{}, []content.ReplacementOption{opt("A")}, content.ReplaceOnce)
			return err
		},
		"rule no options": func() error { _, err := content.NewReplacementRule(1, anyScope, nil, content.ReplaceOnce); return err },
		"rule zero option": func() error {
			_, err := content.NewReplacementRule(1, anyScope, []content.ReplacementOption{{}}, content.ReplaceOnce)
			return err
		},
		"rule repeated option": func() error {
			_, err := content.NewReplacementRule(1, anyScope, []content.ReplacementOption{opt("A"), opt("A")}, content.ReplaceOnce)
			return err
		},
		"rule no limit": func() error {
			_, err := content.NewReplacementRule(1, anyScope, []content.ReplacementOption{opt("A")}, 0)
			return err
		},
		"variant blank requirement": func() error {
			_, err := content.NewMultiattackVariant(" ", []content.AttackSlotGroup{group(1, "A")}, nil, nil)
			return err
		},
		"variant no groups": func() error { _, err := content.NewMultiattackVariant("", nil, nil, nil); return err },
		"variant zero group": func() error {
			_, err := content.NewMultiattackVariant("", []content.AttackSlotGroup{{}}, nil, nil)
			return err
		},
		"variant overlapping groups": func() error {
			_, err := content.NewMultiattackVariant("", []content.AttackSlotGroup{group(1, "Bite"), group(1, "Bite", "Claw")}, nil, nil)
			return err
		},
		"variant scope outside the groups": func() error {
			r := must(content.NewReplacementRule(1, scope("Tail"), []content.ReplacementOption{opt("A")}, content.ReplaceOnce))
			_, err := content.NewMultiattackVariant("", []content.AttackSlotGroup{group(1, "Bite")}, nil, []content.ReplacementRule{r})
			return err
		},
		"variant zero additional use": func() error {
			_, err := content.NewMultiattackVariant("", []content.AttackSlotGroup{group(1, "Bite")}, []content.AdditionalUse{{}}, nil)
			return err
		},
		"variant zero replacement": func() error {
			_, err := content.NewMultiattackVariant("", []content.AttackSlotGroup{group(1, "Bite")}, nil, []content.ReplacementRule{{}})
			return err
		},
		"plan no variants":      func() error { _, err := content.NewMultiattackPlan(nil); return err },
		"plan only conditional": func() error { _, err := content.NewMultiattackPlan([]content.MultiattackVariant{hasten}); return err },
		"plan zero variant":     func() error { _, err := content.NewMultiattackPlan([]content.MultiattackVariant{base, {}}); return err },
	}
	for name, f := range badCases {
		if err := f(); !isInvalidInput(err) {
			t.Errorf("%s: error = %v, want ErrInvalidInput [FR-098]", name, err)
		}
	}
	if _, err := content.NewReplacementRule(2, anyScope, []content.ReplacementOption{withSpell, opt("Sleep Breath")}, content.ReplaceAnyNumber); err != nil {
		t.Errorf("replace two with any number: %v [FR-098]", err)
	}
	for s, want := range map[string]content.ReplacementLimit{"once": content.ReplaceOnce, "anyNumber": content.ReplaceAnyNumber} {
		if got, err := content.ParseReplacementLimit(s); err != nil || got != want || got.String() != s {
			t.Errorf("ParseReplacementLimit(%q) = %v, %v [FR-098]", s, got, err)
		}
	}
	if _, err := content.ParseReplacementLimit("Once"); !isInvalidInput(err) {
		t.Error("ParseReplacementLimit accepted a different case [FR-098]")
	}
	// Name resolution (FR-099) through NewMonster.
	build := func(plan content.MultiattackPlan, actions, bonus, reactions []string) error {
		f := monFields(t)
		mk := func(names []string, withPlan bool) []content.StatBlockPart {
			var out []content.StatBlockPart
			for _, n := range names {
				pf := content.StatBlockPartFields{}
				if withPlan && n == content.MultiattackName {
					pf.Plan = plan
				}
				out = append(out, monPart(t, n, pf))
			}
			return out
		}
		f.Actions, f.BonusActions, f.Reactions = mk(actions, true), mk(bonus, false), mk(reactions, false)
		_, err := monBuild(t, f)
		return err
	}
	slamPlan := must(content.NewMultiattackPlan([]content.MultiattackVariant{base}))
	resolution := []struct {
		name    string
		plan    content.MultiattackPlan
		actions []string
		bonus   []string
		react   []string
		wantOK  bool
	}{
		{"resolves", slamPlan, []string{"Multiattack", "Slam", "Visage"}, nil, nil, true},
		{"group name in a bonus action", slamPlan, []string{"Multiattack", "Visage"}, []string{"Slam"}, nil, false},
		{"unknown name", slamPlan, []string{"Multiattack", "Visage"}, nil, nil, false},
		{"name shared by two Actions parts", slamPlan, []string{"Multiattack", "Slam", "Slam", "Visage"}, nil, nil, false},
		{"group names the Multiattack", must(content.NewMultiattackPlan([]content.MultiattackVariant{variant("", []content.AttackSlotGroup{group(1, "Multiattack")}, nil, nil)})), []string{"Multiattack"}, nil, nil, false},
		{"requiresUsed as a bonus action", plan, []string{"Multiattack", "Slam", "Visage"}, []string{"Hasten"}, nil, true},
		{"requiresUsed as a reaction", plan, []string{"Multiattack", "Slam", "Visage"}, nil, []string{"Hasten"}, true},
		{"requiresUsed as an action", plan, []string{"Multiattack", "Slam", "Visage", "Hasten"}, nil, nil, true},
		{"requiresUsed shared", plan, []string{"Multiattack", "Slam", "Visage"}, []string{"Hasten"}, []string{"Hasten"}, false},
		{"requiresUsed absent", plan, []string{"Multiattack", "Slam", "Visage"}, nil, nil, false},
	}
	for _, c := range resolution {
		err := build(c.plan, c.actions, c.bonus, c.react)
		if (err == nil) != c.wantOK || (err != nil && !isInvalidInput(err)) {
			t.Errorf("%s: NewMonster error = %v, want ok=%v [FR-099]", c.name, err, c.wantOK)
		}
	}
	// A plan outside the Actions is rejected whatever the part is named (FR-036, FR-098).
	for _, place := range []string{"traits", "bonus actions", "reactions", "legendary actions"} {
		pf := content.StatBlockPartFields{Plan: slamPlan}
		mult := []content.StatBlockPart{monPart(t, "Multiattack", pf)}
		fo := monFields(t)
		fo.Actions = []content.StatBlockPart{monPart(t, "Slam", content.StatBlockPartFields{})}
		switch place {
		case "traits":
			fo.Traits = mult
		case "bonus actions":
			fo.BonusActions = mult
		case "reactions":
			fo.Reactions = mult
		default:
			fo.LegendaryActions = must(content.NewLegendaryActions(1, 0, mult))
		}
		if _, err := monBuild(t, fo); !isInvalidInput(err) {
			t.Errorf("a plan in %s: NewMonster error = %v, want ErrInvalidInput [FR-036]", place, err)
		}
	}
	// A nil defence entry is not a value a consumer may receive (FR-024).
	for _, place := range []string{"vulnerabilities", "resistances", "immunities"} {
		fo := monFields(t)
		switch place {
		case "vulnerabilities":
			fo.Vulnerabilities = []content.DefenseEntry{nil}
		case "resistances":
			fo.Resistances = []content.DefenseEntry{nil}
		default:
			fo.Immunities = []content.DefenseEntry{nil}
		}
		if _, err := monBuild(t, fo); !isInvalidInput(err) {
			t.Errorf("a nil entry in %s: NewMonster error = %v, want ErrInvalidInput [FR-024]", place, err)
		}
	}
	// The Multiattack accessor and the plan carried by the part.
	f := monFields(t)
	f.Actions = []content.StatBlockPart{monPart(t, "Slam", content.StatBlockPartFields{}), monPart(t, "Visage", content.StatBlockPartFields{}), monPart(t, "Multiattack", content.StatBlockPartFields{Plan: slamPlan})}
	m := monMake(t, f)
	if p, ok := m.Multiattack(); !ok || p.Name() != "Multiattack" {
		t.Error("Monster.Multiattack [FR-036]")
	}
	if p, _ := m.Multiattack(); !func() bool { pl, ok := p.Plan(); return ok && pl.Equal(slamPlan) }() {
		t.Error("the Multiattack part lost its plan [FR-098]")
	}
	if _, ok := monMake(t, monFields(t)).Multiattack(); ok {
		t.Error("a monster without a Multiattack reports one [FR-036]")
	}
}

// Requirement: monSpec FR-044 - Monster.Spellcasting returns the first part with structured spellcasting, from the
// Traits and then the Actions.
// Path: per case.
func TestMonsterSpellcastingAccessor(t *testing.T) {
	t.Parallel()
	entry := must(content.NewMonsterSpellEntry("Spell", "", 0))
	sc := must(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{
		Ability: ability.Charisma, Groups: []content.MonsterSpellGroup{leveledGroup(t, entry)},
	}))
	f := monFields(t)
	if _, ok := monMake(t, f).Spellcasting(); ok {
		t.Error("a monster without spellcasting reports it [FR-044]")
	}
	f.Actions = []content.StatBlockPart{monPart(t, "Spellcasting", content.StatBlockPartFields{Spellcasting: sc})}
	if got, ok := monMake(t, f).Spellcasting(); !ok || !got.Equal(sc) {
		t.Error("spellcasting in the Actions is not found [FR-044]")
	}
	f.Traits = []content.StatBlockPart{monPart(t, "Plain", content.StatBlockPartFields{}), monPart(t, "Spellcasting", content.StatBlockPartFields{Spellcasting: sc})}
	if got, ok := monMake(t, f).Spellcasting(); !ok || !got.Equal(sc) {
		t.Error("spellcasting in the Traits is not found [FR-044]")
	}
}

// Requirement: monSpec FR-009, FR-095 - a swarm names a valid member size; the limits are the documented constants.
// Path: per case.
func TestSwarmAndLimits(t *testing.T) {
	t.Parallel()
	s := must(content.NewSwarmComposition(content.SizeTiny))
	if s.MemberSize() != content.SizeTiny || s.IsZero() || !(content.SwarmComposition{}).IsZero() {
		t.Error("swarm accessors [FR-009]")
	}
	if _, err := content.NewSwarmComposition(0); !isInvalidInput(err) {
		t.Error("an invalid member size was accepted [FR-009]")
	}
	if content.MonsterMinPage != 258 || content.MonsterMaxPage != 364 || content.MonsterMinSpellLevel != 1 || content.MonsterMaxSpellLevel != 9 ||
		content.MonsterMinRecharge != 1 || content.MonsterMaxRecharge != 6 || content.MonsterSection != "Monsters A-Z" || content.MonsterAnimalsSection != "Animals" {
		t.Error("a limit constant changed [FR-095]")
	}
}

// overLimit returns a number above content.MonsterMaxNumber. Computed at run
// time: on a 32-bit int it wraps to a number below the negative limit, which is
// out of range as well, so the rows that use it hold on both widths.
func overLimit() int {
	limit := content.MonsterMaxNumber
	return limit + 1
}

// Requirement: monSpec FR-001, FR-013 to FR-021, FR-027, FR-032 - the accessors return what the stat block prints.
// Path: happy.
func TestMonsterAccessors(t *testing.T) {
	t.Parallel()
	f := monFields(t)
	f.Alignment, f.Type, f.ArmorClass = content.AlignmentChaoticEvil, content.CreatureTypeFiend, 17
	f.Languages = must(content.NewLanguagesEntry([]string{"Common"}))
	m := monMake(t, f)
	if m.Type() != content.CreatureTypeFiend || m.Alignment() != content.AlignmentChaoticEvil || m.ArmorClass() != 17 {
		t.Errorf("header accessors = %v %v %d", m.Type(), m.Alignment(), m.ArmorClass())
	}
	if m.Initiative() != f.Initiative || m.HitPoints() != f.HitPoints || m.Abilities() != f.Abilities || m.PassivePerception() != 10 {
		t.Error("combat accessors disagree with the fields [FR-014, FR-015, FR-018, FR-027]")
	}
	if m.ChallengeRating() != (content.ChallengeRating{}) || m.ExperiencePoints() != 0 || !m.Languages().Equal(f.Languages) {
		t.Error("detail accessors disagree with the fields [FR-029, FR-030, FR-032]")
	}
	if _, ok := m.ExperiencePointsInLair(); ok {
		t.Error("a monster built without lair XP reports it [FR-032]")
	}
	if len(m.BonusActions())+len(m.Reactions())+len(m.Vulnerabilities())+len(m.Immunities()) != 0 {
		t.Error("an empty collection is not empty [FR-033]")
	}
}

// Requirement: monSpec FR-044 to FR-046 - the exact bounds of the spellcasting numbers, the flags that make a number
// count, and each member that Equal compares.
// Path: boundary.
func TestMonsterSpellcastingBounds(t *testing.T) {
	t.Parallel()
	const maxN = content.MonsterMaxNumber
	for _, n := range []int{1, maxN} {
		if _, err := content.NewSpellFrequencyPerDay(n); err != nil {
			t.Errorf("PerDay(%d) error = %v [FR-045]", n, err)
		}
		if _, err := content.NewSpellFrequencyPerDayEach(n); err != nil {
			t.Errorf("PerDayEach(%d) error = %v [FR-045]", n, err)
		}
	}
	for _, lvl := range []int{1, 9} {
		if _, err := content.NewMonsterSpellEntry("Spell", "", lvl); err != nil {
			t.Errorf("castAtLevel %d error = %v [FR-046]", lvl, err)
		}
	}
	entry := must(content.NewMonsterSpellEntry("Spell", "", 0))
	if _, err := content.NewMonsterSpellGroup(content.SpellFrequencyPerDayEach{}, []content.MonsterSpellEntry{entry}); !isInvalidInput(err) {
		t.Error("a zero N/Day Each frequency was accepted [FR-045]")
	}
	group := must(content.NewMonsterSpellGroup(content.SpellFrequencyAtWill{}, []content.MonsterSpellEntry{entry}))
	base := func() content.MonsterSpellcastingFields {
		return content.MonsterSpellcastingFields{Ability: ability.Wisdom, Groups: []content.MonsterSpellGroup{group}}
	}
	ok := map[string]func(*content.MonsterSpellcastingFields){
		"DC 1":               func(f *content.MonsterSpellcastingFields) { f.SaveDC, f.HasSaveDC = 1, true },
		"DC max":             func(f *content.MonsterSpellcastingFields) { f.SaveDC, f.HasSaveDC = maxN, true },
		"bonus max":          func(f *content.MonsterSpellcastingFields) { f.AttackBonus, f.HasAttackBonus = maxN, true },
		"bonus -max":         func(f *content.MonsterSpellcastingFields) { f.AttackBonus, f.HasAttackBonus = -maxN, true },
		"ignored DC":         func(f *content.MonsterSpellcastingFields) { f.SaveDC = -5 },
		"ignored bonus":      func(f *content.MonsterSpellcastingFields) { f.AttackBonus = overLimit() },
		"ignored DC too big": func(f *content.MonsterSpellcastingFields) { f.SaveDC = overLimit() },
	}
	for name, mutate := range ok {
		f := base()
		mutate(&f)
		s, err := content.NewMonsterSpellcasting(f)
		if err != nil {
			t.Errorf("%s: error = %v [FR-044]", name, err)
			continue
		}
		if dc, has := s.SaveDC(); !has && dc != 0 {
			t.Errorf("%s: an unprinted DC is stored as %d [FR-044]", name, dc)
		}
		if b, has := s.AttackBonus(); !has && b != 0 {
			t.Errorf("%s: an unprinted bonus is stored as %d [FR-044]", name, b)
		}
	}
	bad := map[string]func(*content.MonsterSpellcastingFields){
		"DC above max":    func(f *content.MonsterSpellcastingFields) { f.SaveDC, f.HasSaveDC = overLimit(), true },
		"DC negative":     func(f *content.MonsterSpellcastingFields) { f.SaveDC, f.HasSaveDC = -1, true },
		"bonus below":     func(f *content.MonsterSpellcastingFields) { f.AttackBonus, f.HasAttackBonus = -overLimit(), true },
		"invalid ability": func(f *content.MonsterSpellcastingFields) { f.Ability = ability.Ability(99) },
		"nil frequency group": func(f *content.MonsterSpellcastingFields) {
			f.Groups = []content.MonsterSpellGroup{group, {}}
		},
	}
	for name, mutate := range bad {
		f := base()
		mutate(&f)
		if _, err := content.NewMonsterSpellcasting(f); !isInvalidInput(err) {
			t.Errorf("%s: error = %v [FR-044]", name, err)
		}
	}
	// Equal compares every member.
	a := must(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{
		Ability: ability.Wisdom, SaveDC: 12, HasSaveDC: true, AttackBonus: 4, HasAttackBonus: true, Components: "c", Groups: []content.MonsterSpellGroup{group},
	}))
	vary := map[string]func(*content.MonsterSpellcastingFields){
		"ability":    func(f *content.MonsterSpellcastingFields) { f.Ability = ability.Charisma },
		"dc":         func(f *content.MonsterSpellcastingFields) { f.SaveDC = 13 },
		"has dc":     func(f *content.MonsterSpellcastingFields) { f.HasSaveDC = false },
		"bonus":      func(f *content.MonsterSpellcastingFields) { f.AttackBonus = 5 },
		"has bonus":  func(f *content.MonsterSpellcastingFields) { f.HasAttackBonus = false },
		"components": func(f *content.MonsterSpellcastingFields) { f.Components = "d" },
		"groups":     func(f *content.MonsterSpellcastingFields) { f.Groups = append(f.Groups, group) },
	}
	for name, mutate := range vary {
		f := content.MonsterSpellcastingFields{
			Ability: ability.Wisdom, SaveDC: 12, HasSaveDC: true, AttackBonus: 4, HasAttackBonus: true, Components: "c", Groups: []content.MonsterSpellGroup{group},
		}
		mutate(&f)
		if a.Equal(must(content.NewMonsterSpellcasting(f))) {
			t.Errorf("Equal ignores the %s [FR-044]", name)
		}
	}
	// Spell entries differ by each member.
	e := must(content.NewMonsterSpellEntry("A", "n", 2))
	for _, o := range []content.MonsterSpellEntry{must(content.NewMonsterSpellEntry("B", "n", 2)), must(content.NewMonsterSpellEntry("A", "m", 2)), must(content.NewMonsterSpellEntry("A", "n", 3))} {
		if o == e {
			t.Error("spell entries with a different member compare equal [FR-046]")
		}
	}
}

// Requirement: monSpec FR-002, FR-009, FR-021, FR-027, FR-078 - the exact bounds of the monster's own numbers and
// page, and the section of the source.
// Path: boundary.
func TestNewMonsterBounds(t *testing.T) {
	t.Parallel()
	const maxN = content.MonsterMaxNumber
	cite := func(section string, page int) content.SourceCitation {
		return content.MustSourceCitation("Fixture", section, page)
	}
	build := func(src content.SourceCitation, mutate func(*content.MonsterFields)) error {
		f := monFields(t)
		mutate(&f)
		_, err := content.NewMonster("fixture_beast", "Fixture Beast", src, f)
		return err
	}
	nop := func(*content.MonsterFields) {}
	cases := []struct {
		name   string
		src    content.SourceCitation
		mutate func(*content.MonsterFields)
		ok     bool
	}{
		{"first page", cite(content.MonsterSection, content.MonsterMinPage), nop, true},
		{"last page", cite(content.MonsterAnimalsSection, content.MonsterMaxPage), nop, true},
		{"page before", cite(content.MonsterSection, content.MonsterMinPage-1), nop, false},
		{"page after", cite(content.MonsterSection, content.MonsterMaxPage+1), nop, false},
		{"other section", cite("Elsewhere", 300), nop, false},
		{"no source", content.SourceCitation{}, nop, false},
		{"AC 0", monSource(), func(f *content.MonsterFields) { f.ArmorClass = 0 }, true},
		{"AC max", monSource(), func(f *content.MonsterFields) { f.ArmorClass = maxN }, true},
		{"AC -1", monSource(), func(f *content.MonsterFields) { f.ArmorClass = -1 }, false},
		{"AC above max", monSource(), func(f *content.MonsterFields) { f.ArmorClass = overLimit() }, false},
		{"XP max", monSource(), func(f *content.MonsterFields) { f.ExperiencePoints = maxN }, true},
		{"XP -1", monSource(), func(f *content.MonsterFields) { f.ExperiencePoints = -1 }, false},
		{"XP above max", monSource(), func(f *content.MonsterFields) { f.ExperiencePoints = overLimit() }, false},
		{"lair above XP", monSource(), func(f *content.MonsterFields) { f.ExperiencePoints, f.ExperiencePointsInLair = 5, 6 }, true},
		{"lair equals XP", monSource(), func(f *content.MonsterFields) { f.ExperiencePoints, f.ExperiencePointsInLair = 5, 5 }, false},
		{"lair below XP", monSource(), func(f *content.MonsterFields) { f.ExperiencePoints, f.ExperiencePointsInLair = 5, 4 }, false},
		{"lair above max", monSource(), func(f *content.MonsterFields) { f.ExperiencePointsInLair = overLimit() }, false},
		{"passive perception -max", monSource(), func(f *content.MonsterFields) { f.PassivePerception = -maxN }, true},
		{"passive perception below", monSource(), func(f *content.MonsterFields) { f.PassivePerception = -overLimit() }, false},
		{"passive perception above", monSource(), func(f *content.MonsterFields) { f.PassivePerception = overLimit() }, false},
		{"skill max", monSource(), func(f *content.MonsterFields) { f.Skills = map[content.Skill]int{content.SkillStealth: maxN} }, true},
		{"skill above", monSource(), func(f *content.MonsterFields) { f.Skills = map[content.Skill]int{content.SkillStealth: overLimit()} }, false},
		{"invalid skill", monSource(), func(f *content.MonsterFields) { f.Skills = map[content.Skill]int{0: 1} }, false},
		{"invalid type", monSource(), func(f *content.MonsterFields) { f.Type = 0 }, false},
		{"invalid alignment", monSource(), func(f *content.MonsterFields) { f.Alignment = 0 }, false},
		{"blank tag", monSource(), func(f *content.MonsterFields) { f.Tags = []string{" "} }, false},
		{"blank gear", monSource(), func(f *content.MonsterFields) { f.Gear = []string{""} }, false},
		{"zero sense", monSource(), func(f *content.MonsterFields) { f.Senses = []content.SenseEntry{{}} }, false},
		{"zero hit points", monSource(), func(f *content.MonsterFields) { f.HitPoints = content.HitPointsEntry{} }, false},
		{"zero abilities", monSource(), func(f *content.MonsterFields) { f.Abilities = content.AbilityBlock{} }, false},
		{"zero part", monSource(), func(f *content.MonsterFields) { f.Reactions = []content.StatBlockPart{{}} }, false},
		{"zero speed", monSource(), func(f *content.MonsterFields) { f.Speeds = []content.SpeedEntry{{}} }, false},
		{"duplicate size", monSource(), func(f *content.MonsterFields) {
			f.Sizes = []content.CreatureSize{content.SizeMedium, content.SizeMedium}
		}, false},
		{"invalid size", monSource(), func(f *content.MonsterFields) { f.Sizes = []content.CreatureSize{0} }, false},
		{"no sizes", monSource(), func(f *content.MonsterFields) { f.Sizes = nil }, false},
		{"duplicate speed", monSource(), func(f *content.MonsterFields) {
			f.Speeds = []content.SpeedEntry{monSpeed(t, 30), monSpeed(t, 40)}
		}, false},
	}
	for _, c := range cases {
		err := build(c.src, c.mutate)
		if (err == nil) != c.ok || (err != nil && !isInvalidInput(err)) {
			t.Errorf("%s: NewMonster error = %v, want ok=%v [FR-002]", c.name, err, c.ok)
		}
	}
}
