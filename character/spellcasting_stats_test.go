package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// pbOf is the Proficiency Bonus of a total level.
func pbOf(level int) character.ProficiencyBonus {
	return character.ProficiencyBonusFor(character.MustLevel(level))
}

// modOf is the modifier of an ability score.
func modOf(score int) ability.Modifier { return ability.MustScore(score).Modifier() }

// Requirement: per case.
// Path: per case.
func TestStatsTableB(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		score, level            int
		dc, attack              int
	}{
		{"B1 lowest modifier", "FR-006 Table B1", "unhappy (boundary)", 1, 1, 5, -3},
		{"B2", "FR-006 Table B2", "unhappy (boundary)", 9, 1, 9, 1},
		{"B3 zero modifier", "FR-006 Table B3", "unhappy (boundary)", 10, 1, 10, 2},
		{"B4", "FR-003 FR-004 Table B4", "happy", 16, 1, 13, 5},
		{"B5", "FR-003 FR-004 Table B5", "happy", 16, 5, 14, 6},
		{"B6", "FR-003 FR-004 Table B6", "happy", 18, 9, 16, 8},
		{"B7", "FR-003 FR-004 Table B7", "happy", 20, 17, 19, 11},
		{"B8 highest modifier", "FR-006 Table B8", "unhappy (boundary)", 30, 17, 24, 16},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := character.Stats(character.CasterWizard, modOf(c.score), pbOf(c.level))
			if s.SaveDC().Value() != c.dc || int(s.SaveDC().DifficultyClass()) != c.dc || s.AttackBonus().Value() != c.attack {
				t.Errorf("DC %d attack %d, want %d %d [%s]", s.SaveDC().Value(), s.AttackBonus().Value(), c.dc, c.attack, c.requirement)
			}
			rm := s.AttackRollModifiers()
			pb, ok := rm.ProficiencyBonus()
			if !ok || len(rm.Others()) != 0 || rm.Ability().Value() != modOf(c.score).Value() || pb.Value() != pbOf(c.level).Value() || rm.Total().Value() != c.attack {
				t.Errorf("AttackRollModifiers = %+v, want ability %d + proficiency %d, no others, sum %d", rm, modOf(c.score).Value(), pbOf(c.level).Value(), c.attack)
			}
		})
	}
}

// Requirement: doc/specs/spellcasting.md FR-005, FR-007 - a multiclass character gets
// stats per class from that class's ability, and the stats carry the terms used.
// Path: happy.
func TestStatsCarryTermsPerClass(t *testing.T) {
	t.Parallel()
	pb := pbOf(6)
	w := character.Stats(character.CasterWizard, modOf(18), pb)
	c := character.Stats(character.CasterCleric, modOf(14), pb)
	if w.Ability() != ability.Intelligence || w.AbilityModifier().Value() != 4 || w.ProficiencyBonus() != pb || w.SaveDC().Value() != 15 {
		t.Errorf("wizard stats = %+v", w)
	}
	if c.Ability() != ability.Wisdom || c.AbilityModifier().Value() != 2 || c.SaveDC().Value() != 13 {
		t.Errorf("cleric stats = %+v", c)
	}
}

// Requirement: doc/specs/spellcasting.md FR-145 - StatsForAbility equals Table B for
// every ability and takes the chosen ability, not a class's.
// Path: happy.
func TestStatsForAbility(t *testing.T) {
	t.Parallel()
	for _, a := range ability.All() {
		fa, err := character.NewFeatureSpellAbility(a)
		if err != nil || fa.Ability() != a {
			t.Fatalf("NewFeatureSpellAbility(%v) = %v, %v", a, fa, err)
		}
		got := character.StatsForAbility(fa, modOf(16), pbOf(1))
		if got.Ability() != a || got.SaveDC().Value() != 13 || got.AttackBonus().Value() != 5 {
			t.Errorf("StatsForAbility(%v) = %+v, want DC 13 attack 5", a, got)
		}
	}
}

// Requirement: doc/specs/spellcasting.md FR-145, FR-001 - an invalid ability or class
// is rejected with srderr.ErrInvalidInput or gives the zero stats.
// Path: unhappy (invalid input).
func TestStatsInvalidInput(t *testing.T) {
	t.Parallel()
	if _, err := character.NewFeatureSpellAbility(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewFeatureSpellAbility(0) err = %v", err)
	}
	if got := character.Stats(0, modOf(16), pbOf(1)); got != (character.SpellcastingStats{}) {
		t.Errorf("Stats(invalid class) = %+v, want zero", got)
	}
}
