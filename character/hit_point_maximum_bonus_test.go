package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/derived-stats.md FR-051 - the contribution is 0 below
// grantedAtLevel and flat + perLevel * (n - grantedAtLevel) from it, n being the
// scoped class level or the total level (SRD 5.2.1, "Draconic Resilience", p.
// 69; "Dwarven Toughness", p. 84).
// Path: per case.
func TestHitPointMaximumBonusContribution(t *testing.T) {
	t.Parallel()
	draconic := func() character.HitPointMaximumBonus {
		return character.MustHitPointMaximumBonus(3, 1, 3, "sorcerer")
	}
	cases := []struct {
		name, path string
		bonus      character.HitPointMaximumBonus
		levels     []character.ClassLevel
		want       int
	}{
		{"Draconic at level 3", "happy", draconic(), []character.ClassLevel{cl("sorcerer", 3)}, 3},
		{"Draconic at level 4", "unhappy (boundary)", draconic(), []character.ClassLevel{cl("sorcerer", 4)}, 4},
		{"Draconic below level 3", "unhappy (boundary)", draconic(), []character.ClassLevel{cl("sorcerer", 2)}, 0},
		{"Draconic scoped to Sorcerer", "unhappy (rule interaction)", draconic(), []character.ClassLevel{cl("sorcerer", 3), cl("wizard", 2)}, 3},
		{"Dwarven Toughness level 1", "happy", character.MustHitPointMaximumBonus(1, 1, 1, ""), []character.ClassLevel{cl("fighter", 1)}, 1},
		{"unscoped counts total level", "unhappy (rule interaction)", character.MustHitPointMaximumBonus(1, 1, 1, ""), []character.ClassLevel{cl("fighter", 2), cl("rogue", 2)}, 4},
		{"scope class absent", "unhappy (boundary)", draconic(), []character.ClassLevel{cl("fighter", 3)}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.bonus.ContributionFor(character.MustClassLevels(c.levels)); got != c.want {
				t.Errorf("ContributionFor = %d, want %d", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-029, FR-052, FR-075 - invalid
// fields join one error each in enum order; MustX panics exactly when NewX
// errors.
// Path: per case.
func TestNewHitPointMaximumBonus(t *testing.T) {
	t.Parallel()
	const (
		neg  = character.HitPointMaximumBonusNegativeFlat
		negL = character.HitPointMaximumBonusNegativePerLevel
		out  = character.HitPointMaximumBonusGrantedAtLevelOutOfRange
	)
	cases := []struct {
		name, path    string
		flat, per, at int
		want          []character.HitPointMaximumBonusProblem
	}{
		{"valid", "happy", 3, 1, 3, nil},
		{"zero flat and per level", "unhappy (boundary)", 0, 0, 1, nil},
		{"granted at 20", "unhappy (boundary)", 0, 0, 20, nil},
		{"negative flat", "unhappy (invalid input)", -1, 0, 1, []character.HitPointMaximumBonusProblem{neg}},
		{"negative per level", "unhappy (invalid input)", 0, -1, 1, []character.HitPointMaximumBonusProblem{negL}},
		{"granted at 0", "unhappy (boundary)", 1, 0, 0, []character.HitPointMaximumBonusProblem{out}},
		{"granted at 21", "unhappy (boundary)", 1, 0, 21, []character.HitPointMaximumBonusProblem{out}},
		{"every field invalid", "unhappy (invalid input)", -1, -1, 0, []character.HitPointMaximumBonusProblem{neg, negL, out}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b, err := character.NewHitPointMaximumBonus(c.flat, c.per, c.at, "")
			if c.want == nil {
				if err != nil || b.IsZero() {
					t.Fatalf("NewHitPointMaximumBonus = %v, %v", b, err)
				}
				if got := recovered(t, func() { character.MustHitPointMaximumBonus(c.flat, c.per, c.at, "") }); got != nil {
					t.Errorf("Must panicked: %v", got)
				}
				return
			}
			var got []character.HitPointMaximumBonusProblem
			for _, e := range joined(err) {
				var be *character.HitPointMaximumBonusError
				if !errors.As(e, &be) || !errors.Is(e, srderr.ErrInvalidInput) {
					t.Fatalf("error %v is not a *HitPointMaximumBonusError matching ErrInvalidInput", e)
				}
				got = append(got, be.Problem)
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("problems = %v, want %v", got, c.want)
			}
			if p := recovered(t, func() { character.MustHitPointMaximumBonus(c.flat, c.per, c.at, "") }); !errors.Is(p, srderr.ErrInvalidInput) {
				t.Errorf("Must panic = %v", p)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-051 - the scope is exposed; names
// of problems are stable.
// Path: happy.
func TestHitPointMaximumBonusScopeAndNames(t *testing.T) {
	t.Parallel()
	if got := character.MustHitPointMaximumBonus(1, 1, 1, content.ClassID("wizard")).Scope(); got != "wizard" {
		t.Errorf("Scope = %q", got)
	}
	for p, want := range map[character.HitPointMaximumBonusProblem]string{
		character.HitPointMaximumBonusNegativeFlat: "negativeFlat", character.HitPointMaximumBonusNegativePerLevel: "negativePerLevel",
		character.HitPointMaximumBonusGrantedAtLevelOutOfRange: "grantedAtLevelOutOfRange", 0: "unknown",
	} {
		if p.String() != want {
			t.Errorf("%d.String() = %q", p, p.String())
		}
	}
	e := &character.HitPointMaximumBonusError{Problem: character.HitPointMaximumBonusNegativeFlat}
	if e.Error() == "" || errors.Is(e, &character.HitPointMaximumBonusError{Problem: character.HitPointMaximumBonusNegativePerLevel}) {
		t.Error("error text or Is wrong")
	}
}
