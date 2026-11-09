package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-barbarian.md FR-110, FR-116 - the state is the empty one when initial, compares by value, is refused with a negative
// count (the Must helper panics) and holds its counts.
// Path: per case.
func TestBarbarianStateConstruction(t *testing.T) {
	t.Parallel()
	init := character.InitialBarbarianState()
	if init.RagesExpended() != 0 || init.RelentlessRageUses() != 0 || init.RageActive() || init.PersistentRageUsed() || init.IntimidatingPresenceUsed() || init != (character.BarbarianState{}) {
		t.Errorf("initial state = %#v, want the empty state", init)
	}
	s := character.MustBarbarianState(1, 2)
	if s.RagesExpended() != 1 || s.RelentlessRageUses() != 2 {
		t.Errorf("state = %#v", s)
	}
	for _, c := range []struct {
		name     string
		rages, r int
	}{{"negative Rage count", -1, 0}, {"negative Relentless Rage count", 0, -1}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewBarbarianState(c.rages, c.r)
			wantInvalid(t, err, c.name)
			if got != (character.BarbarianState{}) {
				t.Errorf("state = %#v, want the zero state", got)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Error("MustBarbarianState did not panic")
				}
			}()
			character.MustBarbarianState(c.rages, c.r)
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-110 - BarbarianState implements FeatureState: Equal is by value and false for another type of state; it is stored
// under the one key named barbarian and read back equal, and the creation choices seed the initial state under that key and name no content.
// Path: happy.
func TestBarbarianStateEqualityKeyAndSeed(t *testing.T) {
	t.Parallel()
	a := character.MustBarbarianState(1, 0)
	if !a.Equal(character.MustBarbarianState(1, 0)) || a.Equal(character.MustBarbarianState(2, 0)) || a.Equal(character.MustBarbarianState(1, 1)) ||
		a.Equal(brbRaging(t, 1, 0)) || a.Equal(character.NewWeaponMasteryChoiceState(character.WeaponMasteryChoice{})) {
		t.Error("Equal must compare by value and reject another type")
	}
	if character.BarbarianStateKey().Name() != "barbarian" {
		t.Errorf("key = %q, want barbarian", character.BarbarianStateKey().Name())
	}
	c := brbWith(t, brbBarbarian(t, 3), a)
	if got := brbStateOf(t, c); got != a {
		t.Errorf("stored state = %v, want %v", got, a)
	}
	var choices character.ClassCreationChoices = character.BarbarianCreationChoices{}
	seeded := choices.SeedFeatureStates(character.FeatureStates{})
	s, ok, err := character.ReadFeatureState(seeded, character.BarbarianStateKey())
	if err != nil || !ok || s != character.InitialBarbarianState() || seeded.Len() != 1 {
		t.Errorf("seeded = %v, %v, %v (%d entries)", s, ok, err, seeded.Len())
	}
	if keys := choices.ContentKeys(); keys == nil || len(keys) != 0 {
		t.Errorf("ContentKeys = %v, want an empty slice", keys)
	}
	other := must(character.SetFeatureState(character.FeatureStates{}, character.MustFeatureStateKey[character.ChosenSubclass]("barbarian"), must(character.NewChosenSubclass("x"))))
	if got := choices.SeedFeatureStates(other); !got.Equal(other) {
		t.Error("a foreign state under the key must leave the states unchanged")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-025, FR-026, FR-066, FR-071, FR-082, FR-116 - RagesRemaining is the maximum less the expended count, never below 0 and at
// the maximum of the level; AfterRest on a Short Rest regains one Rage use (never below none expended), resets the Relentless Rage DC and ends a Rage but keeps Persistent
// Rage and Intimidating Presence spent; on a Long Rest regains every use and makes both available; any other kind changes nothing.
// Path: per case (unhappy: boundary).
func TestBarbarianStateAfterRestAndRemaining(t *testing.T) {
	t.Parallel()
	stats := brbStats(t, 3)
	for _, c := range []struct {
		expended, remaining int
	}{{0, 3}, {1, 2}, {3, 0}, {7, 0}} {
		if got := character.MustBarbarianState(c.expended, 0).RagesRemaining(stats); got != c.remaining {
			t.Errorf("RagesRemaining with %d expended = %d, want %d", c.expended, got, c.remaining)
		}
	}
	spent := brbPersistentAndPresenceSpent(t)
	for _, c := range []struct {
		name  string
		start character.BarbarianState
		kind  character.RestKind
		want  character.BarbarianState
	}{
		{"short rest regains one", character.MustBarbarianState(3, 2), character.ShortRest, character.MustBarbarianState(2, 0)},
		{"short rest with none expended", character.MustBarbarianState(0, 1), character.ShortRest, character.MustBarbarianState(0, 0)},
		{"long rest regains all", character.MustBarbarianState(3, 2), character.LongRest, character.MustBarbarianState(0, 0)},
		{"another kind changes nothing", character.MustBarbarianState(3, 2), character.RestKind(0), character.MustBarbarianState(3, 2)},
	} {
		if got := c.start.AfterRest(c.kind); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	short, long := spent.AfterRest(character.ShortRest), spent.AfterRest(character.LongRest)
	if !short.PersistentRageUsed() || !short.IntimidatingPresenceUsed() || short.RageActive() {
		t.Errorf("short rest: %#v, want both still spent and the Rage ended", short)
	}
	if long.PersistentRageUsed() || long.IntimidatingPresenceUsed() || long.RageActive() || long.RagesExpended() != 0 {
		t.Errorf("long rest: %#v, want everything available", long)
	}
	if got := brbRaging(t, 3, 0).AfterRest(character.RestKind(0)); !got.RageActive() {
		t.Error("an unknown rest kind must leave a Rage active")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-066 - the Relentless Rage DC starts at 10 and rises by 5 for each use since the last rest.
// Path: per case.
func TestBarbarianStateRelentlessRageDC(t *testing.T) {
	t.Parallel()
	for uses, want := range []int{10, 15, 20, 25} {
		if got := character.MustBarbarianState(0, uses).RelentlessRageDC(); got != want {
			t.Errorf("DC after %d uses = %d, want %d", uses, got, want)
		}
	}
}

// brbPersistentAndPresenceSpent returns a state with Persistent Rage and Intimidating Presence spent and a Rage active.
func brbPersistentAndPresenceSpent(t *testing.T) character.BarbarianState {
	t.Helper()
	stats := brbStats(t, 15)
	regained := must(character.PersistentRage{}.RegainOnInitiative(brbID(t), stats, brbRaging(t, 15, 1))).(character.RagesRegainedOnInitiative)
	used := must(character.BerserkerRules{}.UseIntimidatingPresence(brbID(t), stats, character.BerserkerSubclassID, regained.State(), character.NewBarbarianTurn(),
		character.BarbarianHostFacts{}, brbMod(t, 16), character.ProficiencyBonusFor(character.MustLevel(15)))).(character.IntimidatingPresenceApplied)
	return used.State()
}
