package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/character-aggregate.md FR-011, FR-012, FR-013, FR-015,
// FR-018 - a created character is level 1 with the supplied values, a maximum of
// the die's top face plus the Constitution modifier, one Hit Die of the class's
// size and one Created event (SRD 5.2.1, "Step 5", p. 22).
// Path: per case.
func TestNewCharacterHitPointMaximum(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, class string
		con         int
		want        int
	}{
		{"Barbarian d12 Con 14", "barbarian", 14, 14}, {"Fighter d10 Con 14", "fighter", 14, 12}, {"Bard d8 Con 10", "bard", 10, 8},
		{"Wizard d6 Con 8", "wizard", 8, 5}, {"Wizard d6 Con 1", "wizard", 1, 1}, {"Barbarian d12 Con 20", "barbarian", 20, 17},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := newParams(t, c.class, scoresOf(t, 10, 10, c.con, 10, 10, 10))
			n, err := character.New(p)
			if err != nil {
				t.Fatal(err)
			}
			ch := n.Character()
			if ch.HitPointMaximum().Value() != c.want || ch.TotalLevel().Value() != 1 || ch.ExperiencePoints().Value() != 0 || ch.HeroicInspiration() != character.NotInspired {
				t.Errorf("FR-011/012: maximum %d level %d xp %d", ch.HitPointMaximum().Value(), ch.TotalLevel().Value(), ch.ExperiencePoints().Value())
			}
			if ch.InitialClass() != p.Class || ch.ID() != p.ID || ch.Species() != p.Species || ch.Background() != p.Background || ch.AbilityScores() != p.AbilityScores || ch.ClassLevels().Len() != 1 {
				t.Error("FR-011: supplied values")
			}
			entries := ch.HitDicePool().Entries()
			if len(entries) != 1 || entries[0].Die != p.HitPointDie || entries[0].Count != 1 {
				t.Errorf("FR-013: pool = %v", entries)
			}
			evs := n.Events()
			created, ok := evs[0].(character.Created)
			if len(evs) != 1 || !ok || created.CharacterID() != p.ID || created.Class() != p.Class || created.Species() != p.Species ||
				created.Background() != p.Background || created.AbilityScores() != p.AbilityScores || created.ExperiencePoints() != ch.ExperiencePoints() || created.HitPointMaximum() != ch.HitPointMaximum() {
				t.Errorf("FR-015: events = %#v", evs)
			}
		})
	}
}

// Requirement: doc/specs/character-aggregate.md FR-012, FR-018 - every
// HitPointDie at Constitution 10 gives a maximum of 6, 8, 10, 12.
// Path: unhappy (boundary).
func TestNewEveryDie(t *testing.T) {
	t.Parallel()
	for die, want := range map[content.HitPointDie]int{content.HitDieD6: 6, content.HitDieD8: 8, content.HitDieD10: 10, content.HitDieD12: 12} {
		p := newParams(t, "wizard", flatScores(t, 10))
		p.HitPointDie = die
		if got := character.MustNew(p).Character().HitPointMaximum().Value(); got != want {
			t.Errorf("%v: %d, want %d", die, got, want)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-014 - a character created with
// the minimum XP for level N can gain levels 2 to N through repeated LevelUp
// without further XP.
// Path: per case.
func TestCreatedAtHigherLevelCanLevelUp(t *testing.T) {
	t.Parallel()
	for _, target := range []int{2, 10, 20} {
		p := newParams(t, "fighter", flatScores(t, 10))
		p.ExperiencePoints = (character.ExperiencePoints{}).MinimumForLevel(character.MustLevel(target))
		c := character.MustNew(p).Character()
		for c.TotalLevel().Value() < target {
			c = changedLevel(t, levelUp(t, c, advancement(t, "fighter", allOf(ability.Strength)), fixedGain(6), nil)).Character()
		}
		if c.TotalLevel().Value() != target || c.IsEligibleForNextLevel() && target < 20 && c.ExperiencePoints().Value() < (character.ExperiencePoints{}).MinimumForLevel(character.MustLevel(target+1)).Value() {
			t.Errorf("target %d: level %d", target, c.TotalLevel().Value())
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-016, FR-082, FR-100 - a score
// above 20 is reported with every offending ability once in declaration order;
// 20 is accepted; New and MustNew agree.
// Path: per case.
func TestNewScoreAboveMaximum(t *testing.T) {
	t.Parallel()
	for _, a := range ability.All() {
		for _, v := range []int{20, 21} {
			s := must(flatScores(t, 10).With(a, ability.MustScore(v)))
			p := newParams(t, "fighter", s)
			_, err := character.New(p)
			panicked := recovered(t, func() { character.MustNew(p) })
			if v == 20 {
				if err != nil || panicked != nil {
					t.Errorf("%v 20: %v", a, err)
				}
				continue
			}
			var prob character.StartingScoreAboveMaximum
			if !errors.As(err, &prob) || !slices.Equal(prob.Abilities(), []ability.Ability{a}) || !errors.Is(err, srderr.ErrInvalidInput) || panicked == nil || panicked.Error() != err.Error() {
				t.Errorf("%v 21: err %v panic %v", a, err, panicked)
			}
		}
	}
	s := scoresOf(t, 22, 10, 25, 10, 10, 30)
	_, err := character.New(newParams(t, "fighter", s))
	var prob character.StartingScoreAboveMaximum
	if !errors.As(err, &prob) || !slices.Equal(prob.Abilities(), []ability.Ability{ability.Strength, ability.Constitution, ability.Charisma}) || len(joined(err)) != 1 {
		t.Errorf("several abilities: %v", err)
	}
	prob.Abilities()[0] = ability.Wisdom
	if prob.Abilities()[0] != ability.Strength || prob.Error() == "" {
		t.Error("Abilities must return a copy")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-100 - a StartingScoreAboveMaximum
// with an empty list is a validation error; abilities are sorted and deduplicated.
// Path: unhappy (invalid input).
func TestStartingScoreAboveMaximumConstructor(t *testing.T) {
	t.Parallel()
	if _, err := character.NewStartingScoreAboveMaximum(nil); !isInvalid(err) {
		t.Errorf("empty: %v", err)
	}
	if _, err := character.NewStartingScoreAboveMaximum([]ability.Ability{ability.Strength, 0}); !isInvalid(err) {
		t.Errorf("invalid ability: %v", err)
	}
	p := character.MustStartingScoreAboveMaximum([]ability.Ability{ability.Charisma, ability.Strength, ability.Charisma})
	if !slices.Equal(p.Abilities(), []ability.Ability{ability.Strength, ability.Charisma}) {
		t.Errorf("abilities = %v", p.Abilities())
	}
	if recovered(t, func() { character.MustStartingScoreAboveMaximum(nil) }) == nil {
		t.Error("Must must panic on an empty list")
	}
	if !errors.Is(p, srderr.ErrInvalidInput) {
		t.Error("the problem must match ErrInvalidInput")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-082 - New reports every
// problem at once, each variant once, in declaration order; an invalid argument
// is a typed problem.
// Path: per case.
func TestNewInvalidArguments(t *testing.T) {
	t.Parallel()
	ok := newParams(t, "fighter", flatScores(t, 10))
	for _, c := range []struct {
		name   string
		mutate func(*character.NewParams)
		arg    character.CreationArgument
	}{
		{"id", func(p *character.NewParams) { p.ID = character.Identifier{} }, character.CreationID},
		{"class", func(p *character.NewParams) { p.Class = "" }, character.CreationClass},
		{"die", func(p *character.NewParams) { p.HitPointDie = 0 }, character.CreationHitPointDie},
		{"species", func(p *character.NewParams) { p.Species = "" }, character.CreationSpecies},
		{"background", func(p *character.NewParams) { p.Background = "" }, character.CreationBackground},
		{"scores", func(p *character.NewParams) { p.AbilityScores = ability.Scores{} }, character.CreationAbilityScores},
		{"inspiration", func(p *character.NewParams) { p.HeroicInspiration = 9 }, character.CreationHeroicInspiration},
	} {
		p := ok
		c.mutate(&p)
		_, err := character.New(p)
		var got character.InvalidCreationArgument
		if !errors.As(err, &got) || got.Argument != c.arg || !errors.Is(err, srderr.ErrInvalidInput) || got.Error() == "" || len(joined(err)) != 1 {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	p := ok
	p.ID, p.Class, p.AbilityScores = character.Identifier{}, "", scoresOf(t, 21, 10, 10, 10, 10, 10)
	_, err := character.New(p)
	problems := joined(err)
	if len(problems) != 3 {
		t.Fatalf("problems = %v", problems)
	}
	var first, second character.InvalidCreationArgument
	var third character.StartingScoreAboveMaximum
	if !errors.As(problems[0], &first) || !errors.As(problems[1], &second) || first.Argument != character.CreationID || second.Argument != character.CreationClass {
		t.Errorf("order = %v", problems)
	}
	if !errors.As(problems[2], &third) {
		t.Errorf("the score problem comes after the argument problems: %v", problems)
	}
	if character.CreationArgument(0).String() != "unknown" || character.CreationAbilityScores.String() != "abilityScores" || character.CreationArgument(99).String() != "unknown" {
		t.Error("argument strings")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-011, FR-089, FR-092, FR-094 -
// New keeps a supplied starting XP, Heroic Inspiration and feature states; the
// default ClassCreationChoices seeds nothing and names no keys.
// Path: happy.
func TestNewKeepsSuppliedValues(t *testing.T) {
	t.Parallel()
	rage := character.MustFeatureStateKey[rageState]("rage")
	var choices character.ClassCreationChoices = character.NoClassCreationChoices{}
	if len(choices.ContentKeys()) != 0 {
		t.Error("NoClassCreationChoices names no keys")
	}
	seeded := choices.SeedFeatureStates(must(character.SetFeatureState(character.FeatureStates{}, rage, rageState{2})))
	p := newParams(t, "fighter", flatScores(t, 10))
	p.ExperiencePoints, p.HeroicInspiration, p.FeatureStates = character.MustExperiencePoints(300), character.Inspired, seeded
	c := character.MustNew(p).Character()
	if v, ok, _ := character.ReadFeatureState(c.FeatureStates(), rage); !ok || v.uses != 2 || c.ExperiencePoints().Value() != 300 || c.HeroicInspiration() != character.Inspired {
		t.Error("supplied values must be kept")
	}
}
