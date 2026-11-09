package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

const (
	bright = content.LightBright
	dim    = content.LightDim
	dark   = content.LightDarkness
)

// Requirement: doc/specs/exploration.md FR-027 to FR-032, cases 4.5a - the level a source
// alone gives, with the radius edge inside, a Cone needing the host's report and a
// radius ignoring it (SRD 5.2.1, "Light", p. 11; "Torch", p. 100; "Lantern, Bullseye", p. 98).
// Path: per case.
func TestLevelAt(t *testing.T) {
	t.Parallel()
	src := character.SrdLightSources{}
	type at struct {
		d    int
		cone bool
		want content.LightLevel
	}
	cases := []struct {
		name, path string
		emission   character.LightEmission
		rows       []at
	}{
		{"Torch", "unhappy (boundary)", src.Torch(), []at{{0, false, bright}, {20, false, bright}, {21, false, dim}, {40, false, dim}, {41, false, dark}}},
		{"Candle", "unhappy (boundary)", src.Candle(), []at{{5, false, bright}, {6, false, dim}, {10, false, dim}, {11, false, dark}}},
		{"Lamp", "unhappy (boundary)", src.Lamp(), []at{{15, false, bright}, {16, false, dim}, {45, false, dim}, {46, false, dark}}},
		{"hood lowered", "unhappy (boundary)", src.HoodedLantern(character.HoodLowered), []at{{0, false, dim}, {5, false, dim}, {6, false, dark}}},
		{"hood raised", "unhappy (boundary)", src.HoodedLantern(character.HoodRaised), []at{{30, false, bright}, {31, false, dim}, {60, false, dim}, {61, false, dark}}},
		{"Bullseye inside the Cone", "unhappy (boundary)", src.BullseyeLantern(), []at{{60, true, bright}, {61, true, dim}, {120, true, dim}, {121, true, dark}}},
		{"Bullseye outside the Cone", "unhappy (rule interaction)", src.BullseyeLantern(), []at{{0, false, dark}, {5, false, dark}, {100, false, dark}}},
		{"radius ignores withinCone", "unhappy (rule interaction)", src.Torch(), []at{{10, true, bright}, {10, false, bright}}},
		{"only Bright Light", "unhappy (boundary)", character.MustLightEmission(character.LightShapeRadius, 10, 0), []at{{10, false, bright}, {11, false, dark}}},
		{"only Dim Light, distance 0", "unhappy (boundary)", character.MustLightEmission(character.LightShapeRadius, 0, 10), []at{{0, false, dim}, {10, false, dim}, {11, false, dark}}},
		{"zero emission", "unhappy (invalid input)", character.LightEmission{}, []at{{0, false, dark}, {0, true, dark}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, r := range c.rows {
				space, err := character.NewLightSpace(r.d, r.cone)
				if err != nil {
					t.Fatal(err)
				}
				if got := (character.LightSourceRules{}).LevelAt(c.emission, space); got != r.want {
					t.Errorf("LevelAt(d %d, cone %v) = %v, want %v", r.d, r.cone, got, r.want)
				}
			}
		})
	}
}

// Requirement: doc/specs/exploration.md FR-033, cases 4.5b - the brightest level wins
// regardless of order; none gives Darkness (SRD 5.2.1, "Light", p. 11).
// Path: per case.
func TestStrongest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		levels     []content.LightLevel
		want       content.LightLevel
	}{
		{"none", "unhappy (empty and minimal)", nil, dark},
		{"one source", "happy", []content.LightLevel{dim}, dim},
		{"ambient dim and a torch", "unhappy (rule interaction)", []content.LightLevel{dim, bright}, bright},
		{"ambient bright and a candle", "unhappy (rule interaction)", []content.LightLevel{bright, dim}, bright},
		{"two dim", "unhappy (rule interaction)", []content.LightLevel{dim, dim}, dim},
		{"dark and dim", "unhappy (rule interaction)", []content.LightLevel{dark, dim}, dim},
		{"mixed", "unhappy (determinism)", []content.LightLevel{dark, bright, dim}, bright},
		{"mixed reversed", "unhappy (determinism)", []content.LightLevel{dim, bright, dark}, bright},
		{"invalid ignored", "unhappy (invalid input)", []content.LightLevel{0, 99}, dark},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := slices.Clone(c.levels)
			if got := (character.LightSourceRules{}).Strongest(c.levels); got != c.want {
				t.Errorf("Strongest(%v) = %v, want %v", c.levels, got, c.want)
			}
			if !slices.Equal(in, c.levels) {
				t.Error("Strongest changed its input")
			}
		})
	}
}

// Requirement: doc/specs/exploration.md FR-034, FR-035, table 4.5c - SrdLightSources holds
// exactly the six printed emissions (SRD 5.2.1, "Candle", p. 96; "Lamp", "Lantern,
// Bullseye", "Lantern, Hooded", p. 98; "Torch", p. 100).
// Path: per case.
func TestSrdLightSources(t *testing.T) {
	t.Parallel()
	src := character.SrdLightSources{}
	want := []character.LightEmission{
		character.MustLightEmission(character.LightShapeRadius, 5, 5),
		character.MustLightEmission(character.LightShapeRadius, 15, 30),
		character.MustLightEmission(character.LightShapeCone, 60, 60),
		character.MustLightEmission(character.LightShapeRadius, 30, 30),
		character.MustLightEmission(character.LightShapeRadius, 0, 5),
		character.MustLightEmission(character.LightShapeRadius, 20, 20),
	}
	got := src.All()
	if !slices.Equal(got, want) {
		t.Fatalf("All() = %v, want %v", got, want)
	}
	got[0] = character.LightEmission{} // a copy
	if src.All()[0] != want[0] {
		t.Error("All must return a fresh slice")
	}
	if !src.HoodedLantern(character.HoodedLanternHood(0)).IsZero() {
		t.Error("an invalid hood must give the zero emission")
	}
}
