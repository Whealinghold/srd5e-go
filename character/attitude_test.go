package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/social-interaction.md FR-001 - Attitude has exactly Friendly,
// Indifferent and Hostile, in that order; the zero value and values past Hostile are
// invalid; MonsterDefaultAttitude equals Indifferent (SRD 5.2.1, "Attitude", p. 177;
// "Indifferent [Attitude]", p. 184).
// Path: per case.
func TestAttitude(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		a                       character.Attitude
		valid                   bool
		str                     string
	}{
		{"zero", "FR-001", "unhappy (invalid input)", 0, false, "unknown"},
		{"friendly", "FR-001", "happy", character.Friendly, true, "friendly"},
		{"indifferent", "FR-001", "happy", character.Indifferent, true, "indifferent"},
		{"hostile", "FR-001", "happy", character.Hostile, true, "hostile"},
		{"past hostile", "FR-001", "unhappy (boundary)", character.Hostile + 1, false, "unknown"},
		{"negative", "FR-001", "unhappy (boundary)", -1, false, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.a.Valid() != c.valid || c.a.String() != c.str {
				t.Errorf("Attitude(%d) = valid %v, %q; want %v, %q [%s, %s]", int(c.a), c.a.Valid(), c.a.String(), c.valid, c.str, c.requirement, c.path)
			}
		})
	}
	if character.Friendly != 1 || character.Indifferent != 2 || character.Hostile != 3 || character.MonsterDefaultAttitude != character.Indifferent {
		t.Errorf("order or default wrong: %d %d %d, default %v", character.Friendly, character.Indifferent, character.Hostile, character.MonsterDefaultAttitude)
	}
}
