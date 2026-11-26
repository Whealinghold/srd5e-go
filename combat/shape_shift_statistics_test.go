package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/shape-changing-spells.md FR-003 and table B - StatisticsOf returns the retained statistics of the spell and mode; every other statistic is replaced.
// Path: per case.
func TestStatisticsOf(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		spell                   combat.ShapeShiftSpell
		mode                    combat.TrueTransformation
		want                    []combat.RetainedStatistic
	}{
		{"polymorph", `SRD 5.2.1, "Polymorph", p. 153`, "happy", combat.PolymorphSpell, 0,
			[]combat.RetainedStatistic{combat.RetainedAlignment, combat.RetainedPersonality, combat.RetainedCreatureType, combat.RetainedHitPoints, combat.RetainedHitPointDice}},
		{"true polymorph creature into creature", `SRD 5.2.1, "True Polymorph", p. 170`, "happy", combat.TruePolymorphSpell, combat.CreatureIntoCreature,
			[]combat.RetainedStatistic{combat.RetainedAlignment, combat.RetainedPersonality, combat.RetainedHitPoints, combat.RetainedHitPointDice}},
		{"true polymorph object into creature", `SRD 5.2.1, "True Polymorph", p. 170`, "unhappy (rule interaction)", combat.TruePolymorphSpell, combat.ObjectIntoCreature, nil},
		{"true polymorph creature into object", `SRD 5.2.1, "True Polymorph", p. 170`, "unhappy (rule interaction)", combat.TruePolymorphSpell, combat.CreatureIntoObject, nil},
		{"shapechange", `SRD 5.2.1, "Shapechange", p. 161`, "happy", combat.ShapechangeSpell, 0,
			[]combat.RetainedStatistic{combat.RetainedAlignment, combat.RetainedPersonality, combat.RetainedCreatureType, combat.RetainedHitPoints, combat.RetainedHitPointDice,
				combat.RetainedMentalScores, combat.RetainedProficiencies, combat.RetainedAbilityToCommunicate, combat.RetainedSpellcastingFeature}},
		{"animal shapes", `SRD 5.2.1, "Animal Shapes", p. 108`, "happy", combat.AnimalShapesSpell, 0,
			[]combat.RetainedStatistic{combat.RetainedAlignment, combat.RetainedCreatureType, combat.RetainedHitPoints, combat.RetainedHitPointDice,
				combat.RetainedMentalScores, combat.RetainedAbilityToCommunicate}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.ShapeShiftSpells{}).StatisticsOf(c.spell, c.mode)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got.Retained(), c.want) || !got.ReplacesEveryOther() {
				t.Errorf("StatisticsOf(%v, %v) retains %v, want %v [%s]", c.spell, c.mode, got.Retained(), c.want, c.requirement)
			}
			for _, s := range c.want {
				if !got.Keeps(s) {
					t.Errorf("Keeps(%v) = false", s)
				}
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-003 - the returned sets are copies and equal by value.
// Path: unhappy (immutability).
func TestStatisticsOfReturnsCopiesEqualByValue(t *testing.T) {
	t.Parallel()
	a, _ := (combat.ShapeShiftSpells{}).StatisticsOf(combat.PolymorphSpell, 0)
	b, _ := (combat.ShapeShiftSpells{}).StatisticsOf(combat.PolymorphSpell, 0)
	list := a.Retained()
	list[0] = combat.RetainedProficiencies
	if a != b || a.Keeps(combat.RetainedProficiencies) || a.Retained()[0] != combat.RetainedAlignment {
		t.Error("changing a returned list changed the set, or equal sets differ")
	}
	other, _ := (combat.ShapeShiftSpells{}).StatisticsOf(combat.ShapechangeSpell, 0)
	if a == other {
		t.Error("the sets of two spells are equal")
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-003, FR-004 - an invalid spell, and True Polymorph without a mode, return an error and no value.
// Path: unhappy (invalid input).
func TestStatisticsAndRestrictionsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	s := combat.ShapeShiftSpells{}
	if _, err := s.StatisticsOf(0, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("StatisticsOf(zero spell) error = %v", err)
	}
	if _, err := s.StatisticsOf(combat.TruePolymorphSpell, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("StatisticsOf(True Polymorph, no mode) error = %v", err)
	}
	if _, err := s.RestrictionsOf(0, 0, false); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("RestrictionsOf(zero spell) error = %v", err)
	}
	if _, err := s.RestrictionsOf(combat.TruePolymorphSpell, 0, false); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("RestrictionsOf(True Polymorph, no mode) error = %v", err)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-004, FR-005 and table B - RestrictionsOf returns what the form forbids; Shapechange states none of the three limits, and its Spellcasting feature applies only when held (SRD 5.2.1, "Polymorph", p. 154; "Shapechange", p. 161; "Animal Shapes", p. 108; "True Polymorph", p. 170).
// Path: per case.
func TestRestrictionsOf(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		spell                   combat.ShapeShiftSpell
		mode                    combat.TrueTransformation
		holds                   bool
		want                    combat.FormRestrictions
	}{
		{"polymorph", "FR-004", "unhappy (rule interaction)", combat.PolymorphSpell, 0, false, combat.FormRestrictions{ActionsLimitedByAnatomy: true, Speech: combat.FormCannotSpeak, Spellcasting: combat.FormCannotCast}},
		{"true polymorph creature into creature", "FR-004", "unhappy (rule interaction)", combat.TruePolymorphSpell, combat.CreatureIntoCreature, false,
			combat.FormRestrictions{ActionsLimitedByAnatomy: true, Speech: combat.FormCannotSpeak, Spellcasting: combat.FormCannotCast}},
		{"true polymorph object mode", "FR-004 n/a", "unhappy (rule interaction)", combat.TruePolymorphSpell, combat.ObjectIntoCreature, false, combat.FormRestrictions{}},
		{"true polymorph creature into object", "FR-004 n/a", "unhappy (rule interaction)", combat.TruePolymorphSpell, combat.CreatureIntoObject, false, combat.FormRestrictions{}},
		{"animal shapes", "FR-004", "unhappy (rule interaction)", combat.AnimalShapesSpell, 0, false,
			combat.FormRestrictions{ActionsLimitedByAnatomy: true, Speech: combat.FormAbilityToCommunicateRetained, Spellcasting: combat.FormCannotCast}},
		{"shapechange with the feature", "FR-004 no limits, FR-005", "unhappy (boundary)", combat.ShapechangeSpell, 0, true,
			combat.FormRestrictions{Speech: combat.FormAbilityToCommunicateRetained, Spellcasting: combat.FormSpellcastingFeatureRetained}},
		{"shapechange without the feature", "FR-005", "unhappy (boundary)", combat.ShapechangeSpell, 0, false,
			combat.FormRestrictions{Speech: combat.FormAbilityToCommunicateRetained, Spellcasting: combat.FormNoSpellcastingFeature}},
		{"the fact is ignored for polymorph", "FR-005", "unhappy (boundary)", combat.PolymorphSpell, 0, true,
			combat.FormRestrictions{ActionsLimitedByAnatomy: true, Speech: combat.FormCannotSpeak, Spellcasting: combat.FormCannotCast}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.ShapeShiftSpells{}).RestrictionsOf(c.spell, c.mode, c.holds)
			if err != nil || got != c.want {
				t.Errorf("RestrictionsOf(%v, %v, %v) = %+v, %v, want %+v [%s]", c.spell, c.mode, c.holds, got, err, c.want, c.requirement)
			}
		})
	}
}
