package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var noReasons []combat.FormIneligibility

// ssSized is a Beast form of cr printing the given sizes, with size chosen.
func ssSized(tb testing.TB, kind content.CreatureType, cr string, chosen content.CreatureSize, printed ...content.CreatureSize) combat.CreatureForm {
	tb.Helper()
	return luMust(combat.NewCreatureForm(ssFacts(tb, kind, cr, 10, printed...), chosen))
}

// Requirement: per case (table E1 of doc/specs/shape-changing-spells.md; each case is one row or one half of a row)
// Path: per case.
func TestEligibility(t *testing.T) {
	t.Parallel()
	beast, human, dragon := content.CreatureTypeBeast, content.CreatureTypeHumanoid, content.CreatureTypeDragon
	med, large, huge := content.SizeMedium, content.SizeLarge, content.SizeHuge
	r := func(rs ...combat.FormIneligibility) []combat.FormIneligibility { return rs }
	cases := []struct {
		name, requirement, path string
		q                       combat.FormQuery
		want                    []combat.FormIneligibility
	}{
		{"level 1 allows CR 1", "FR-008", "unhappy (boundary)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssLevel(1), Form: ssBeast(t, "1", 5)}, noReasons},
		{"level 1 allows CR 1/4", "FR-008", "unhappy (boundary)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssLevel(1), Form: ssBeast(t, "1/4", 5)}, noReasons},
		{"level 1 refuses CR 2", "FR-008", "unhappy (boundary)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssLevel(1), Form: ssBeast(t, "2", 5)}, r(combat.ChallengeRatingTooHigh)},
		{"CR 0 target allows CR 0", "FR-008", "unhappy (boundary)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssCR("0"), Form: ssBeast(t, "0", 5)}, noReasons},
		{"CR 0 target refuses CR 1/8", "FR-008", "unhappy (boundary)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssCR("0"), Form: ssBeast(t, "1/8", 5)}, r(combat.ChallengeRatingTooHigh)},
		{"Polymorph refuses a non-Beast", "FR-009", "unhappy (invalid input)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssCR("5"), Form: ssForm(t, human, "1", 5, med)}, r(combat.NotABeast)},
		{"Polymorph two reasons", "FR-016", "unhappy (rule interaction)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssLevel(1), Form: ssForm(t, dragon, "5", 5, med)}, r(combat.NotABeast, combat.ChallengeRatingTooHigh)},
		{"Animal Shapes CR 4", "FR-010", "unhappy (boundary)", combat.FormQuery{Spell: combat.AnimalShapesSpell, Form: ssForm(t, beast, "4", 5, large)}, noReasons},
		{"Animal Shapes CR 5", "FR-010", "unhappy (boundary)", combat.FormQuery{Spell: combat.AnimalShapesSpell, Form: ssForm(t, beast, "5", 5, large)}, r(combat.ChallengeRatingTooHigh)},
		{"Animal Shapes Large", "FR-010", "unhappy (boundary)", combat.FormQuery{Spell: combat.AnimalShapesSpell, Form: ssForm(t, beast, "1", 5, large)}, noReasons},
		{"Animal Shapes Huge", "FR-010", "unhappy (boundary)", combat.FormQuery{Spell: combat.AnimalShapesSpell, Form: ssForm(t, beast, "1", 5, huge)}, r(combat.SizeTooLarge)},
		{"Animal Shapes judges the chosen size, Large of Huge or Large", "FR-017", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.AnimalShapesSpell, Form: ssSized(t, beast, "1", large, huge, large)}, noReasons},
		{"Animal Shapes judges the chosen size, Huge of Huge or Large", "FR-017", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.AnimalShapesSpell, Form: ssSized(t, beast, "1", huge, huge, large)}, r(combat.SizeTooLarge)},
		{"Animal Shapes refuses a non-Beast", "FR-010", "unhappy (invalid input)", combat.FormQuery{Spell: combat.AnimalShapesSpell, Form: ssForm(t, human, "1", 5, med)}, r(combat.NotABeast)},
		{"Shapechange refuses a Construct", "FR-011", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Rating: ssCR("10"), HasSeenKind: true, Form: ssForm(t, content.CreatureTypeConstruct, "1", 5, med)}, r(combat.ConstructOrUndead)},
		{"Shapechange refuses an Undead", "FR-011", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Rating: ssLevel(12), HasSeenKind: true, Form: ssForm(t, content.CreatureTypeUndead, "3", 5, med)}, r(combat.ConstructOrUndead)},
		{"Shapechange allows a Dragon at the caster's level", "FR-011", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Rating: ssLevel(17), HasSeenKind: true, Form: ssForm(t, dragon, "17", 5, huge)}, noReasons},
		{"Shapechange refuses a Dragon above it", "FR-011", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Rating: ssLevel(17), HasSeenKind: true, Form: ssForm(t, dragon, "18", 5, huge)}, r(combat.ChallengeRatingTooHigh)},
		{"Shapechange refuses a kind not seen", "FR-011", "unhappy (state error)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Rating: ssLevel(17), Form: ssForm(t, dragon, "1", 5, huge)}, r(combat.KindNotSeen)},
		{"stat block Shapechange allows a Beast", "FR-012", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Terms: combat.StatBlockShapechange, Rating: ssCR("13"), HasSeenKind: true, Form: ssForm(t, beast, "1", 5, med)}, noReasons},
		{"stat block Shapechange allows a Humanoid", "FR-012", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Terms: combat.StatBlockShapechange, Rating: ssCR("13"), HasSeenKind: true, Form: ssForm(t, human, "1", 5, med)}, noReasons},
		{"stat block Shapechange refuses a Dragon", "FR-012", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Terms: combat.StatBlockShapechange, Rating: ssCR("13"), HasSeenKind: true, Form: ssForm(t, dragon, "1", 5, med)}, r(combat.NotBeastOrHumanoid)},
		{"stat block Shapechange refuses a Construct for both reasons", "FR-012", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Terms: combat.StatBlockShapechange, Rating: ssCR("13"), HasSeenKind: true, Form: ssForm(t, content.CreatureTypeConstruct, "1", 5, med)},
			r(combat.ConstructOrUndead, combat.NotBeastOrHumanoid)},
		{"Shapechange every reason in order", "FR-016", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.ShapechangeSpell, Terms: combat.StatBlockShapechange, Rating: ssCR("1"), Form: ssForm(t, content.CreatureTypeUndead, "5", 5, med)},
			r(combat.ConstructOrUndead, combat.NotBeastOrHumanoid, combat.ChallengeRatingTooHigh, combat.KindNotSeen)},
		{"True Polymorph allows a Construct", "FR-013", "happy",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.CreatureIntoCreature, Rating: ssLevel(20), Form: ssForm(t, content.CreatureTypeConstruct, "20", 5, med)}, noReasons},
		{"True Polymorph creature at its rating", "FR-013", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.CreatureIntoCreature, Rating: ssCR("5"), Form: ssBeast(t, "5", 5)}, noReasons},
		{"True Polymorph creature above its rating", "FR-013", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.CreatureIntoCreature, Rating: ssCR("5"), Form: ssBeast(t, "6", 5)}, r(combat.ChallengeRatingTooHigh)},
		{"object into a Large creature", "FR-014", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.ObjectIntoCreature, Object: combat.ObjectFacts{Size: med}, Form: ssForm(t, beast, "1", 5, large)}, r(combat.SizeTooLarge)},
		{"object into an equal creature", "FR-014", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.ObjectIntoCreature, Object: combat.ObjectFacts{Size: med}, Form: ssBeast(t, "1", 5)}, noReasons},
		{"object into CR 9", "FR-014", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.ObjectIntoCreature, Object: combat.ObjectFacts{Size: huge}, Form: ssForm(t, beast, "9", 5, huge)}, noReasons},
		{"object into CR 10", "FR-014", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.ObjectIntoCreature, Object: combat.ObjectFacts{Size: huge}, Form: ssForm(t, beast, "10", 5, huge)}, r(combat.ChallengeRatingTooHigh)},
		{"a magical object", "FR-014", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.ObjectIntoCreature, Object: combat.ObjectFacts{Size: med, IsMagical: true}, Form: ssBeast(t, "1", 5)}, r(combat.ObjectIsMagical)},
		{"a worn object", "FR-014", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.ObjectIntoCreature, Object: combat.ObjectFacts{Size: med, IsWornOrCarried: true}, Form: ssBeast(t, "1", 5)}, r(combat.ObjectWornOrCarried)},
		{"an object with every reason", "FR-016", "unhappy (rule interaction)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.ObjectIntoCreature, Object: combat.ObjectFacts{Size: med, IsMagical: true, IsWornOrCarried: true}, Form: ssForm(t, beast, "12", 5, large)},
			r(combat.SizeTooLarge, combat.ChallengeRatingTooHigh, combat.ObjectIsMagical, combat.ObjectWornOrCarried)},
		{"creature into a larger object", "FR-015", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.CreatureIntoObject, TargetSize: med, Form: ssObject(t, large)}, r(combat.SizeTooLarge)},
		{"creature into an equal object", "FR-015", "unhappy (boundary)",
			combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.CreatureIntoObject, TargetSize: med, Form: ssObject(t, med)}, noReasons},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.ShapeShiftSpells{}).Eligibility(c.q)
			if err != nil {
				t.Fatalf("Eligibility error = %v", err)
			}
			if len(c.want) == 0 {
				if _, ok := got.(combat.FormEligible); !ok {
					t.Errorf("Eligibility = %#v, want FormEligible [%s]", got, c.requirement)
				}
				return
			}
			bad, ok := got.(combat.FormIneligible)
			if !ok || !slices.Equal(bad.Reasons(), c.want) || bad.String() != "form_ineligible" {
				t.Errorf("Eligibility = %#v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-016 - the reasons of a FormIneligible are a copy, never empty, and in the fixed order of the enum.
// Path: unhappy (immutability).
func TestFormIneligibleReasonsAreACopy(t *testing.T) {
	t.Parallel()
	got, err := (combat.ShapeShiftSpells{}).Eligibility(combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssLevel(1), Form: ssForm(t, content.CreatureTypeDragon, "5", 5, content.SizeMedium)})
	if err != nil {
		t.Fatal(err)
	}
	bad := got.(combat.FormIneligible)
	list := bad.Reasons()
	list[0] = combat.KindNotSeen
	if again := bad.Reasons(); len(again) == 0 || again[0] != combat.NotABeast {
		t.Errorf("Reasons() = %v after a returned list was changed", again)
	}
	for i := combat.NotABeast; i <= combat.ObjectWornOrCarried; i++ {
		if i.String() == "unknown" {
			t.Errorf("reason %d has no name", i)
		}
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-008, FR-055 - a query that does not fit the spell returns an error matching ErrInvalidInput and no answer: no spell, no form, an object form for Polymorph, a creature form for an object mode, True Polymorph without a mode, a missing or zero rating, an object or creature without a size, an invalid terms value.
// Path: per case.
func TestEligibilityRejectsInvalidQueries(t *testing.T) {
	t.Parallel()
	med := content.SizeMedium
	cases := []struct {
		name, requirement, path string
		q                       combat.FormQuery
	}{
		{"no spell", "FR-055", "unhappy (invalid input)", combat.FormQuery{Rating: ssLevel(1), Form: ssBeast(t, "1", 5)}},
		{"no form", "FR-055", "unhappy (invalid input)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssLevel(1)}},
		{"an object form for Polymorph", "FR-054", "unhappy (invalid input)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssLevel(1), Form: ssObject(t, med)}},
		{"the zero creature form", "FR-007", "unhappy (invalid input)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: ssLevel(1), Form: combat.CreatureForm{}}},
		{"no rating", "FR-008", "unhappy (invalid input)", combat.FormQuery{Spell: combat.PolymorphSpell, Form: ssBeast(t, "1", 5)}},
		{"the zero level", "FR-008", "unhappy (invalid input)", combat.FormQuery{Spell: combat.PolymorphSpell, Rating: combat.RatedByLevel{}, Form: ssBeast(t, "1", 5)}},
		{"True Polymorph without a mode", "FR-013", "unhappy (invalid input)", combat.FormQuery{Spell: combat.TruePolymorphSpell, Rating: ssLevel(1), Form: ssBeast(t, "1", 5)}},
		{"a creature form for creature into object", "FR-015", "unhappy (invalid input)", combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.CreatureIntoObject, TargetSize: med, Form: ssBeast(t, "1", 5)}},
		{"an object form for creature into creature", "FR-013", "unhappy (invalid input)", combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.CreatureIntoCreature, Rating: ssLevel(1), Form: ssObject(t, med)}},
		{"an object without a size", "FR-014", "unhappy (invalid input)", combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.ObjectIntoCreature, Form: ssBeast(t, "1", 5)}},
		{"a creature without a size", "FR-015", "unhappy (invalid input)", combat.FormQuery{Spell: combat.TruePolymorphSpell, Mode: combat.CreatureIntoObject, Form: ssObject(t, med)}},
		{"invalid terms", "FR-012", "unhappy (invalid input)", combat.FormQuery{Spell: combat.ShapechangeSpell, Terms: 7, Rating: ssLevel(1), Form: ssBeast(t, "1", 5)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.ShapeShiftSpells{}).Eligibility(c.q)
			if got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("Eligibility = %#v, %v, want nil and ErrInvalidInput [%s]", got, err, c.requirement)
			}
		})
	}
}
