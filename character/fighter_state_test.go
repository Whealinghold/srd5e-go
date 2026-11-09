package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-fighter.md FR-145, FR-151, FR-153 - the state is the empty one when initial, compares by value, is
// refused with a negative expended count or an invalid marker (the Must helper panics) and holds its counts and markers.
// Path: per case.
func TestFighterStateConstruction(t *testing.T) {
	t.Parallel()
	init := character.InitialFighterState()
	if init.SecondWindExpended() != 0 || init.ActionSurgeExpended() != 0 || init.IndomitableExpended() != 0 || init.FightingStyle() != "" || init.AdditionalFightingStyle() != "" || init != (character.FighterState{}) {
		t.Errorf("initial state = %#v, want the empty state", init)
	}
	s := character.MustFighterState(1, 2, 3, "defense", "archery")
	if s.SecondWindExpended() != 1 || s.ActionSurgeExpended() != 2 || s.IndomitableExpended() != 3 || s.FightingStyle() != "defense" || s.AdditionalFightingStyle() != "archery" {
		t.Errorf("state = %#v", s)
	}
	for _, c := range []struct {
		name              string
		sw, as, ind       int
		style, additional content.FeatID
	}{
		{"negative Second Wind count", -1, 0, 0, "", ""}, {"negative Action Surge count", 0, -1, 0, "", ""},
		{"negative Indomitable count", 0, 0, -1, "", ""}, {"invalid style marker", 0, 0, 0, "Not Valid", ""},
		{"invalid additional marker", 0, 0, 0, "", "Not Valid"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewFighterState(c.sw, c.as, c.ind, c.style, c.additional)
			wantInvalid(t, err, c.name)
			if got != (character.FighterState{}) {
				t.Errorf("state = %#v, want the zero state", got)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Error("MustFighterState did not panic")
				}
			}()
			character.MustFighterState(c.sw, c.as, c.ind, c.style, c.additional)
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-145 - FighterState implements FeatureState: Equal is by value and false for another
// type of state; it is stored under the one key named fighter and read back equal.
// Path: happy.
func TestFighterStateEqualityAndKey(t *testing.T) {
	t.Parallel()
	a, b := character.MustFighterState(1, 0, 0, "defense", ""), character.MustFighterState(1, 0, 0, "defense", "")
	if !a.Equal(b) || a.Equal(character.MustFighterState(2, 0, 0, "defense", "")) || a.Equal(character.MustFighterState(1, 0, 0, "archery", "")) ||
		a.Equal(character.MustFighterState(1, 0, 0, "defense", "archery")) || a.Equal(character.NewWeaponMasteryChoiceState(character.WeaponMasteryChoice{})) {
		t.Error("Equal must compare by value and reject another type")
	}
	if character.FighterStateKey().Name() != "fighter" {
		t.Errorf("key = %q, want fighter", character.FighterStateKey().Name())
	}
	c := fgtWith(t, fgtFighter(t, 3), a)
	if got := fgtStateOf(t, c); got != a {
		t.Errorf("stored state = %#v, want %#v", got, a)
	}
}

// Requirement: doc/specs/class-fighter.md FR-062, FR-076, FR-153 - the remaining uses are the maximum less the expended count and never
// below 0: level 3 with 2 expended Second Wind uses has 1 remaining at level 4; a count above the maximum leaves 0; a use expended at
// level 16 stays expended at 17.
// Path: per case (unhappy: boundary).
func TestFighterStateRemainingUses(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name                                 string
		level                                int
		state                                character.FighterState
		secondWind, actionSurge, indomitable int
	}{
		{"level 3 with 2 Second Wind expended", 3, fgtState(2, 0, 0), 0, 1, 0},
		{"level 4 with 2 Second Wind expended", 4, fgtState(2, 0, 0), 1, 1, 0},
		{"a count above the maximum", 3, fgtState(9, 9, 9), 0, 0, 0},
		{"Action Surge at 17 with one expended", 17, fgtState(0, 1, 0), 4, 1, 3},
		{"Action Surge at 16 with one expended", 16, fgtState(0, 1, 0), 4, 0, 2},
		{"Indomitable at level 9 with one expended", 9, fgtState(0, 0, 1), 3, 1, 0},
		{"Indomitable at level 13 with one expended", 13, fgtState(0, 0, 1), 4, 1, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			stats := fgtStats(t, c.level)
			if got := c.state.SecondWindRemaining(stats); got != c.secondWind {
				t.Errorf("Second Wind remaining = %d, want %d", got, c.secondWind)
			}
			if got := c.state.ActionSurgeRemaining(stats); got != c.actionSurge {
				t.Errorf("Action Surge remaining = %d, want %d", got, c.actionSurge)
			}
			if got := c.state.IndomitableRemaining(stats); got != c.indomitable {
				t.Errorf("Indomitable remaining = %d, want %d", got, c.indomitable)
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-063, FR-074, FR-083, FR-148 - a Short Rest regains one expended Second Wind use (none if
// none is expended) and all Action Surge uses and no Indomitable use; a Long Rest regains all uses of all three; the markers stay; any
// other kind changes nothing.
// Path: per case.
func TestFighterStateAfterRest(t *testing.T) {
	t.Parallel()
	full := character.MustFighterState(3, 2, 3, "defense", "archery")
	for _, c := range []struct {
		name string
		in   character.FighterState
		kind character.RestKind
		want character.FighterState
	}{
		{"short rest", full, character.ShortRest, character.MustFighterState(2, 0, 3, "defense", "archery")},
		{"short rest with none expended", character.MustFighterState(0, 1, 0, "", ""), character.ShortRest, character.MustFighterState(0, 0, 0, "", "")},
		{"long rest", full, character.LongRest, character.MustFighterState(0, 0, 0, "defense", "archery")},
		{"an invalid kind", full, character.RestKind(0), full},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.in.AfterRest(c.kind); got != c.want {
				t.Errorf("AfterRest = %#v, want %#v", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-144, FR-151 - FighterCreationChoices seeds exactly the initial FighterState under the key
// fighter, leaves the other keys and its argument unchanged, names no content (an empty slice, a fresh one each call) and satisfies
// ClassCreationChoices; a foreign state under the key is left as it is.
// Path: happy / unhappy (invalid input).
func TestFighterCreationChoices(t *testing.T) {
	t.Parallel()
	other := must(character.SetFeatureState(character.FeatureStates{}, character.MustFeatureStateKey[character.ChosenSubclass]("subclass_wizard"), must(character.NewChosenSubclass("evoker"))))
	var choices character.ClassCreationChoices = character.FighterCreationChoices{}
	seeded := choices.SeedFeatureStates(other)
	got, ok, err := character.ReadFeatureState(seeded, character.FighterStateKey())
	if err != nil || !ok || got != character.InitialFighterState() || seeded.Len() != 2 || other.Len() != 1 {
		t.Errorf("seeded = %v (len %d, original %d), state %v %v %v", seeded, seeded.Len(), other.Len(), got, ok, err)
	}
	keys := choices.ContentKeys()
	if keys == nil || len(keys) != 0 {
		t.Errorf("ContentKeys = %#v, want an empty non-nil slice", keys)
	}
	foreign := must(character.SetFeatureState(character.FeatureStates{}, character.MustFeatureStateKey[character.ChosenSubclass]("fighter"), must(character.NewChosenSubclass("champion"))))
	if got := choices.SeedFeatureStates(foreign); !got.Equal(foreign) {
		t.Errorf("a foreign state under fighter was replaced: %v", got)
	}
}
