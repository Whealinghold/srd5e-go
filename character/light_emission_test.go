package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/exploration.md FR-026, FR-050, cases 4.5a - an emission needs a
// valid shape, non-negative distances and at least one of them above zero (SRD 5.2.1,
// "Torch", p. 100).
// Path: per case.
func TestNewLightEmission(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path  string
		shape       character.LightShape
		bright, dim int
		wantProblem character.ExplorationInput // 0 means accepted
	}{
		{"torch", "happy", character.LightShapeRadius, 20, 20, 0},
		{"only Dim Light", "unhappy (boundary)", character.LightShapeRadius, 0, 10, 0},
		{"only Bright Light", "unhappy (boundary)", character.LightShapeRadius, 10, 0, 0},
		{"a cone", "happy", character.LightShapeCone, 60, 60, 0},
		{"no light", "unhappy (invalid input)", character.LightShapeRadius, 0, 0, character.ExplorationNoLight},
		{"negative bright", "unhappy (invalid input)", character.LightShapeRadius, -1, 5, character.ExplorationLightRadius},
		{"negative dim", "unhappy (invalid input)", character.LightShapeRadius, 5, -1, character.ExplorationLightRadius},
		{"invalid shape", "unhappy (invalid input)", character.LightShape(0), 5, 5, character.ExplorationLightShape},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, err := character.NewLightEmission(c.shape, c.bright, c.dim)
			if c.wantProblem != 0 {
				if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.ExplorationError{Input: c.wantProblem}) || !e.IsZero() {
					t.Errorf("err = %v", err)
				}
				if recovered(t, func() { character.MustLightEmission(c.shape, c.bright, c.dim) }) == nil {
					t.Error("MustLightEmission must panic")
				}
				return
			}
			if err != nil || e.Shape() != c.shape || e.BrightFeet() != c.bright || e.DimFeet() != c.dim || e.IsZero() {
				t.Errorf("emission = %+v, %v", e, err)
			}
		})
	}
	if character.LightShapeRadius.String() != "radius" || character.LightShapeCone.String() != "cone" || character.LightShape(0).String() != "unknown" {
		t.Error("LightShape strings")
	}
}

// Requirement: doc/specs/exploration.md FR-030, cases 4.5a - a distance of 0 is the
// source's own space; a negative distance is rejected.
// Path: per case.
func TestNewLightSpace(t *testing.T) {
	t.Parallel()
	s, err := character.NewLightSpace(0, true)
	if err != nil || s.DistanceFeet() != 0 || !s.WithinCone() {
		t.Errorf("distance 0 = %+v, %v", s, err)
	}
	if _, err := character.NewLightSpace(-1, false); !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.ExplorationError{Input: character.ExplorationLightDistance}) {
		t.Errorf("distance -1 err = %v", err)
	}
}
