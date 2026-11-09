package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-bard.md FR-025, FR-121 (SRD 5.2.1, "Number of Uses", p. 32) - the Number of Uses is the Charisma modifier with a minimum of once: modifiers -5, -1 and 0 give 1, +1 gives 1, +2 gives 2 and +5 gives 5; the
// initial pool is full.
// Path: per case (unhappy: boundary).
func TestInitialBardicInspirationPool(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ score, want int }{{1, 1}, {8, 1}, {10, 1}, {12, 1}, {14, 2}, {20, 5}} {
		p := character.InitialBardicInspirationPool(brdMod(t, c.score))
		if p.Maximum() != c.want || p.Remaining() != c.want {
			t.Errorf("Charisma %d: pool %d of %d, want %d of %d", c.score, p.Remaining(), p.Maximum(), c.want, c.want)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-045, FR-121 - a pool enforces 0 <= remaining <= maximum with a maximum of at least 1, and WithMaximumFrom clamps the remaining uses to a lower maximum and never raises them when the maximum grows.
// Path: per case (unhappy: invalid input, boundary).
func TestBardicInspirationPoolInvariantAndMaximum(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		max, rem int
		ok       bool
	}{{3, 3, true}, {3, 0, true}, {1, 1, true}, {3, 4, false}, {3, -1, false}, {0, 0, false}, {-1, 0, false}} {
		p, err := character.NewBardicInspirationPool(c.max, c.rem)
		if (err == nil) != c.ok || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) || (c.ok && (p.Maximum() != c.max || p.Remaining() != c.rem)) {
			t.Errorf("NewBardicInspirationPool(%d, %d) = %v, %v", c.max, c.rem, p, err)
		}
	}
	p := must(character.NewBardicInspirationPool(4, 3))
	if got := p.WithMaximumFrom(brdMod(t, 10)); got.Maximum() != 1 || got.Remaining() != 1 {
		t.Errorf("shrunk = %d of %d, want 1 of 1", got.Remaining(), got.Maximum())
	}
	if got := p.WithMaximumFrom(brdMod(t, 20)); got.Maximum() != 5 || got.Remaining() != 3 {
		t.Errorf("grown = %d of %d, want 3 of 5 (never raised)", got.Remaining(), got.Maximum())
	}
}

// Requirement: doc/specs/class-bard.md FR-118 - the pool is a FeatureState under one typed key named bardic_inspiration_pool; Equal compares by value and rejects another state type.
// Path: happy / unhappy (invalid input).
func TestBardicInspirationPoolIsAFeatureState(t *testing.T) {
	t.Parallel()
	a, b := must(character.NewBardicInspirationPool(3, 2)), must(character.NewBardicInspirationPool(3, 1))
	if !a.Equal(a) || a.Equal(b) || a.Equal(character.ClericResources{}) {
		t.Errorf("Equal: a=a %v, a=b %v, a=other type %v", a.Equal(a), a.Equal(b), a.Equal(character.ClericResources{}))
	}
	if character.BardicInspirationPoolKey().Name() != "bardic_inspiration_pool" {
		t.Errorf("key = %q", character.BardicInspirationPoolKey().Name())
	}
}
