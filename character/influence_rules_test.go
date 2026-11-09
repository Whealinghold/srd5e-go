package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/social-interaction.md FR-003, FR-004, FR-005 - Friendly gives
// exactly one Advantage, Hostile exactly one Disadvantage, Indifferent (and an invalid
// attitude) none, as a non-nil empty slice; the slice is a fresh copy (SRD 5.2.1, "Friendly
// [Attitude]", p. 182; "Hostile [Attitude]", p. 183; "Indifferent [Attitude]", p. 184).
// Path: per case.
func TestAttitudeCircumstances(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		a                       character.Attitude
		want                    []modifier.RollCircumstance
	}{
		{"friendly", "FR-003", "happy", character.Friendly, []modifier.RollCircumstance{modifier.Advantage{}}},
		{"hostile", "FR-004", "happy", character.Hostile, []modifier.RollCircumstance{modifier.Disadvantage{}}},
		{"indifferent", "FR-005", "unhappy (empty and minimal)", character.Indifferent, nil},
		{"zero", "FR-005", "unhappy (invalid input)", 0, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.AttitudeCircumstances(c.a)
			if got == nil || len(got) != len(c.want) {
				t.Fatalf("circumstances = %#v, want %#v [%s, %s]", got, c.want, c.requirement, c.path)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("circumstances[%d] = %#v, want %#v", i, got[i], c.want[i])
				}
			}
			if len(got) > 0 {
				got[0] = nil
				if again := character.AttitudeCircumstances(c.a); again[0] == nil {
					t.Error("the returned slice aliases shared state")
				}
			}
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-008 - IsSuggested is true exactly for the
// rows of the Influence Checks table (SRD 5.2.1, p. 184); an invalid approach is never
// suggested.
// Path: per case.
func TestIsSuggested(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		approach                character.InfluenceApproach
		comp                    character.InfluenceComprehension
		ct                      content.CreatureType
		want                    bool
	}{
		{"intimidate, any target", "FR-008", "unhappy (boundary)", character.Intimidate, character.DoesNotUnderstand, content.CreatureTypeAberration, true},
		{"amuse, any target", "FR-008", "happy", character.Amuse, character.DoesNotUnderstand, content.CreatureTypeBeast, true},
		{"deceive, understands", "FR-008", "happy", character.Deceive, character.Understands, content.CreatureTypeHumanoid, true},
		{"deceive, does not understand", "FR-008", "unhappy (boundary)", character.Deceive, character.DoesNotUnderstand, content.CreatureTypeHumanoid, false},
		{"persuade, understands", "FR-008", "happy", character.Persuade, character.Understands, content.CreatureTypeGiant, true},
		{"persuade, does not understand", "FR-008", "unhappy (boundary)", character.Persuade, character.DoesNotUnderstand, content.CreatureTypeGiant, false},
		{"coax a Beast", "FR-008", "happy", character.Coax, character.DoesNotUnderstand, content.CreatureTypeBeast, true},
		{"coax a Monstrosity", "FR-008", "happy", character.Coax, character.Understands, content.CreatureTypeMonstrosity, true},
		{"coax a Humanoid", "FR-008", "unhappy (boundary)", character.Coax, character.Understands, content.CreatureTypeHumanoid, false},
		{"coax an Ooze", "FR-008", "unhappy (boundary)", character.Coax, character.DoesNotUnderstand, content.CreatureTypeOoze, false},
		{"invalid approach", "FR-008", "unhappy (invalid input)", 0, character.Understands, content.CreatureTypeBeast, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			target := must(character.NewInfluenceTarget(score(t, 10), c.ct, c.comp))
			if got := character.IsSuggested(c.approach, target); got != c.want {
				t.Errorf("IsSuggested = %v, want %v [%s, %s]", got, c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-010 - the default DC is 15 or the
// Intelligence score, whichever is higher (SRD 5.2.1, "Influence [Action]", p. 184).
// Path: per case.
func TestDefaultDC(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		score                   int
		want                    d20.DifficultyClass
	}{
		{"lowest score", "FR-010", "unhappy (boundary)", 1, 15},
		{"below the floor", "FR-010", "unhappy (boundary)", 14, 15},
		{"at the floor", "FR-010", "unhappy (boundary)", 15, 15},
		{"one above the floor", "FR-010", "unhappy (boundary)", 16, 16},
		{"typical high", "FR-010", "happy", 20, 20},
		{"highest score", "FR-010", "unhappy (boundary)", 30, 30},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := character.DefaultDC(score(t, c.score)); got != c.want {
				t.Errorf("DefaultDC(%d) = %d, want %d [%s, %s]", c.score, got, c.want, c.requirement, c.path)
			}
		})
	}
}
