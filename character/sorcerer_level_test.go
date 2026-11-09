package character_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-sorcerer.md FR-018 - a Sorcerer level below 1 or above 20 is rejected with srderr.ErrInvalidInput, 1 and 20 are accepted, and the zero value is invalid.
// Path: per case (boundary).
func TestSorcererLevelBounds(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n    int
		ok   bool
		path string
	}{{-1, false, "invalid"}, {0, false, "boundary"}, {1, true, "boundary"}, {20, true, "boundary"}, {21, false, "boundary"}} {
		t.Run(strconv.Itoa(c.n), func(t *testing.T) {
			t.Parallel()
			l, err := character.NewSorcererLevel(c.n)
			if c.ok != (err == nil) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) || (c.ok && (l.Value() != c.n || l.IsZero())) {
				t.Errorf("NewSorcererLevel(%d) = %v, %v; want ok %v", c.n, l, err, c.ok)
			}
		})
	}
	if zero := (character.SorcererLevel{}); !zero.IsZero() || zero.Value() != 0 {
		t.Errorf("zero level = %v, want invalid", zero)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-018 - MustSorcererLevel panics with the error of NewSorcererLevel, and only then.
// Path: unhappy (invalid input).
func TestMustSorcererLevel(t *testing.T) {
	t.Parallel()
	if got := character.MustSorcererLevel(7).Value(); got != 7 {
		t.Errorf("MustSorcererLevel(7) = %d", got)
	}
	defer func() {
		err, _ := recover().(error)
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an error matching ErrInvalidInput", err)
		}
	}()
	character.MustSorcererLevel(0)
}

// Requirement: doc/specs/class-sorcerer.md FR-047, FR-055, Table P (SRD 5.2.1, "Sorcerer Features", p. 65; "Level 2: Metamagic", p. 66) - the Sorcery Points maximum is none at level 1 and the level
// from 2 to 20, and the Metamagic options known are 0 at level 1, 2 from 2, 4 from 10 and 6 from 17.
// Path: per level (boundaries 1, 2, 9, 10, 16, 17, 20).
func TestSorcererLevelFigures(t *testing.T) {
	t.Parallel()
	entitled := []int{0, 2, 2, 2, 2, 2, 2, 2, 2, 4, 4, 4, 4, 4, 4, 4, 6, 6, 6, 6}
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			wantPoints := n
			if n == 1 {
				wantPoints = 0
			}
			l := character.MustSorcererLevel(n)
			if l.SorceryPointsMaximum() != wantPoints || l.MetamagicEntitled() != entitled[n-1] {
				t.Errorf("level %d = %d points, %d options; want %d and %d", n, l.SorceryPointsMaximum(), l.MetamagicEntitled(), wantPoints, entitled[n-1])
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-137 - a profile needs a level and a valid subclass id; the zero profile is invalid.
// Path: unhappy (invalid input).
func TestNewSorcererProfile(t *testing.T) {
	t.Parallel()
	if _, err := character.NewSorcererProfile(character.SorcererLevel{}, ""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero level err = %v, want ErrInvalidInput", err)
	}
	if _, err := character.NewSorcererProfile(character.MustSorcererLevel(3), "Not Valid"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("bad subclass err = %v, want ErrInvalidInput", err)
	}
	p := sorDraconic(6)
	if p.Level().Value() != 6 || p.Subclass() != character.DraconicSorcerySubclassID {
		t.Errorf("profile = %d %q", p.Level().Value(), p.Subclass())
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-137 - the profile is built from ClassLevels and the ChosenSubclass; a character with no Sorcerer level has none, and another type of state under the subclass key is an error.
// Path: per case (happy / unhappy: not a Sorcerer, bad state).
func TestSorcererProfileOf(t *testing.T) {
	t.Parallel()
	fighter := restored(t, 355000, 200, flatScores(t, 10), cl("fighter", 5))
	if _, ok, err := character.SorcererProfileOf(fighter); ok || err != nil {
		t.Errorf("fighter = (%v, %v), want no profile", ok, err)
	}
	plain := sorChar(t, 4, 16, "", sorFresh(sorProfile(4, "")))
	if p, ok, err := character.SorcererProfileOf(plain); err != nil || !ok || p.Level().Value() != 4 || p.Subclass() != "" {
		t.Errorf("plain = (%v, %v, %v)", p, ok, err)
	}
	draconic := sorChar(t, 6, 16, "draconic_sorcery", sorFresh(sorProfile(6, "")))
	if p, ok, err := character.SorcererProfileOf(draconic); err != nil || !ok || p.Level().Value() != 6 || p.Subclass() != "draconic_sorcery" {
		t.Errorf("draconic = (%v, %v, %v)", p, ok, err)
	}
	wrong := must(character.WithFeatureState(plain, character.MustFeatureStateKey[character.SorcererState]("subclass_sorcerer"), character.SorcererState{}))
	if _, _, err := character.SorcererProfileOf(wrong); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("wrong state err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-019 (SRD 5.2.1, "Sorcerer Features", p. 65) - level 1 has 0 Sorcery Points and no Font of Magic or Metamagic; they are available from level 2, Sorcerous Restoration from 5, Sorcery Incarnate from 7 and Arcane Apotheosis at 20.
// Path: per case (boundary).
func TestSorcererFeaturesAt(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level int
		want  character.SorcererFeatureAvailability
	}{
		{1, character.SorcererFeatureAvailability{}},
		{2, character.SorcererFeatureAvailability{SorceryPoints: 2, FontOfMagic: true, Metamagic: true}},
		{4, character.SorcererFeatureAvailability{SorceryPoints: 4, FontOfMagic: true, Metamagic: true}},
		{5, character.SorcererFeatureAvailability{SorceryPoints: 5, FontOfMagic: true, Metamagic: true, SorcerousRestoration: true}},
		{6, character.SorcererFeatureAvailability{SorceryPoints: 6, FontOfMagic: true, Metamagic: true, SorcerousRestoration: true}},
		{7, character.SorcererFeatureAvailability{SorceryPoints: 7, FontOfMagic: true, Metamagic: true, SorcerousRestoration: true, SorceryIncarnate: true}},
		{19, character.SorcererFeatureAvailability{SorceryPoints: 19, FontOfMagic: true, Metamagic: true, SorcerousRestoration: true, SorceryIncarnate: true}},
		{20, character.SorcererFeatureAvailability{SorceryPoints: 20, FontOfMagic: true, Metamagic: true, SorcerousRestoration: true, SorceryIncarnate: true, ArcaneApotheosis: true}},
	} {
		t.Run("level "+strconv.Itoa(c.level), func(t *testing.T) {
			t.Parallel()
			if got := character.SorcererFeaturesAt(sorProfile(c.level, "")); got != c.want {
				t.Errorf("SorcererFeaturesAt(%d) = %+v, want %+v", c.level, got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-024 (SRD 5.2.1, "Level 3: Sorcerer Subclass", p. 66) - the subclass features at or below the level are returned (Draconic Sorcery: 3, 6, 14, 18; Draconic Spells at 3 too) and none above it or while no subclass is chosen.
// Path: per case (boundary).
func TestSubclassFeaturesAvailable(t *testing.T) {
	t.Parallel()
	sub := sorDef(t).Subclasses()[0]
	res := func(ids ...string) []content.FeatureID {
		out := []content.FeatureID{}
		for _, id := range ids {
			out = append(out, content.FeatureID("draconic_sorcery_"+id))
		}
		return out
	}
	for _, c := range []struct {
		name     string
		profile  character.SorcererProfile
		expected []content.FeatureID
	}{
		{"level 2", sorDraconic(2), res()},
		{"level 3", sorDraconic(3), res("draconic_resilience", "draconic_spells")},
		{"level 5", sorDraconic(5), res("draconic_resilience", "draconic_spells")},
		{"level 6", sorDraconic(6), res("draconic_resilience", "draconic_spells", "elemental_affinity")},
		{"level 13", sorDraconic(13), res("draconic_resilience", "draconic_spells", "elemental_affinity")},
		{"level 14", sorDraconic(14), res("draconic_resilience", "draconic_spells", "elemental_affinity", "dragon_wings")},
		{"level 17", sorDraconic(17), res("draconic_resilience", "draconic_spells", "elemental_affinity", "dragon_wings")},
		{"level 18", sorDraconic(18), res("draconic_resilience", "draconic_spells", "elemental_affinity", "dragon_wings", "dragon_companion")},
		{"no subclass chosen", sorProfile(18, ""), res()},
		{"another subclass", sorProfile(18, "wild_magic"), res()},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := character.SubclassFeaturesAvailable(c.profile, sub); !slices.Equal(got, c.expected) {
				t.Errorf("SubclassFeaturesAvailable = %v, want %v", got, c.expected)
			}
		})
	}
}
