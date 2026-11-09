package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/social-interaction.md FR-002 - a target built without an attitude
// is Indifferent and equals one with an explicit Indifferent; the accessors return what was
// given (SRD 5.2.1, "Indifferent [Attitude]", p. 184).
// Path: happy.
func TestNewInfluenceTargetDefaultsToIndifferent(t *testing.T) {
	t.Parallel()
	def, err := character.NewInfluenceTarget(score(t, 12), content.CreatureTypeBeast, character.Understands)
	explicit, err2 := character.NewInfluenceTarget(score(t, 12), content.CreatureTypeBeast, character.Understands, character.WithAttitude(character.Indifferent))
	if err != nil || err2 != nil || def != explicit || def.Attitude() != character.Indifferent || !def.Valid() {
		t.Fatalf("default = %#v, %v; explicit = %#v, %v", def, err, explicit, err2)
	}
	if len(character.AttitudeCircumstances(def.Attitude())) != 0 {
		t.Errorf("default target contributes %v", character.AttitudeCircumstances(def.Attitude()))
	}
	if def.Intelligence() != score(t, 12) || def.CreatureType() != content.CreatureTypeBeast || def.Comprehension() != character.Understands {
		t.Errorf("accessors = %v, %v, %v", def.Intelligence(), def.CreatureType(), def.Comprehension())
	}
}

// Requirement: doc/specs/social-interaction.md FR-029 - targets are equal when every field
// is equal and unequal when any single field differs.
// Path: per case.
func TestInfluenceTargetEquality(t *testing.T) {
	t.Parallel()
	base := func(s int, ct content.CreatureType, c character.InfluenceComprehension, a character.Attitude) character.InfluenceTarget {
		return must(character.NewInfluenceTarget(score(t, s), ct, c, character.WithAttitude(a)))
	}
	ref := base(10, content.CreatureTypeGiant, character.Understands, character.Friendly)
	cases := []struct {
		name, requirement, path string
		other                   character.InfluenceTarget
		equal                   bool
	}{
		{"same", "FR-029", "happy", base(10, content.CreatureTypeGiant, character.Understands, character.Friendly), true},
		{"score", "FR-029", "unhappy (any differing field)", base(11, content.CreatureTypeGiant, character.Understands, character.Friendly), false},
		{"creature type", "FR-029", "unhappy (any differing field)", base(10, content.CreatureTypeBeast, character.Understands, character.Friendly), false},
		{"comprehension", "FR-029", "unhappy (any differing field)", base(10, content.CreatureTypeGiant, character.DoesNotUnderstand, character.Friendly), false},
		{"attitude", "FR-029", "unhappy (any differing field)", base(10, content.CreatureTypeGiant, character.Understands, character.Hostile), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if (ref == c.other) != c.equal {
				t.Errorf("equal = %v, want %v [%s, %s]", ref == c.other, c.equal, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-030, FR-002 - the constructor rejects a
// zero Intelligence score, an invalid creature type, comprehension or attitude with
// ErrInvalidInput and returns the zero (invalid) target.
// Path: per case.
func TestNewInfluenceTargetRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		score                   ability.Score
		ct                      content.CreatureType
		comp                    character.InfluenceComprehension
		att                     character.Attitude
	}{
		{"zero score", "FR-030", "unhappy (boundary)", ability.Score{}, content.CreatureTypeBeast, character.Understands, character.Friendly},
		{"zero creature type", "FR-030", "unhappy (invalid input)", score(t, 10), 0, character.Understands, character.Friendly},
		{"zero comprehension", "FR-030", "unhappy (invalid input)", score(t, 10), content.CreatureTypeBeast, 0, character.Friendly},
		{"zero attitude set explicitly", "FR-002", "unhappy (invalid input)", score(t, 10), content.CreatureTypeBeast, character.Understands, 0},
		{"attitude past hostile", "FR-002", "unhappy (boundary)", score(t, 10), content.CreatureTypeBeast, character.Understands, character.Hostile + 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewInfluenceTarget(c.score, c.ct, c.comp, character.WithAttitude(c.att))
			if !errors.Is(err, srderr.ErrInvalidInput) || got != (character.InfluenceTarget{}) || got.Valid() {
				t.Errorf("got %#v, %v [%s, %s]", got, err, c.requirement, c.path)
			}
		})
	}
}
