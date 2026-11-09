package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-monk.md FR-132, FR-133, FR-150 - the state is the empty one when initial, compares by value, is refused with a
// negative expended count (the Must helper panics) and holds its counts, flag and vibrations; it stores no level and no maximum.
// Path: per case.
func TestMonkStateConstruction(t *testing.T) {
	t.Parallel()
	init := character.InitialMonkState()
	if init.FocusExpended() != 0 || init.UncannyMetabolismUsed() || init.WholenessOfBodySpent() != 0 || init.Vibrations().Active() || init != (character.MonkState{}) {
		t.Errorf("initial state = %#v, want the empty state", init)
	}
	s := character.MustMonkState(2, true, 1, mnkVibrations(9))
	if s.FocusExpended() != 2 || !s.UncannyMetabolismUsed() || s.WholenessOfBodySpent() != 1 || s.Vibrations().DurationDays() != 9 || s.Vibrations().Target().Ref() != "orc-1" {
		t.Errorf("state = %#v", s)
	}
	for _, c := range []struct {
		name             string
		focus, wholeness int
	}{{"negative Focus Points count", -1, 0}, {"negative Wholeness of Body count", 0, -1}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewMonkState(c.focus, false, c.wholeness, character.QuiveringPalmVibrations{})
			wantInvalid(t, err, c.name)
			if got != (character.MonkState{}) {
				t.Errorf("state = %#v, want the zero state", got)
			}
			defer func() {
				if recover() == nil {
					t.Error("MustMonkState did not panic")
				}
			}()
			character.MustMonkState(c.focus, false, c.wholeness, character.QuiveringPalmVibrations{})
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-116 - a target reference is 1 to 128 bytes and vibrations need a target and at least 1 day; the
// zero values are no target and no vibrations.
// Path: per case (unhappy: boundary, invalid input).
func TestQuiveringPalmValues(t *testing.T) {
	t.Parallel()
	for _, ref := range []string{"", string(make([]byte, 129))} {
		_, err := character.NewQuiveringPalmTarget(ref)
		wantInvalid(t, err, "target reference of length "+string(rune('0'+len(ref)%10)))
	}
	if tg, err := character.NewQuiveringPalmTarget(string(make([]byte, 128))); err != nil || tg.IsZero() {
		t.Errorf("a 128-byte reference = %v, %v, want accepted", tg, err)
	}
	if !(character.QuiveringPalmTarget{}).IsZero() || (character.QuiveringPalmVibrations{}).Active() {
		t.Error("the zero target and vibrations must be empty")
	}
	_, err := character.NewQuiveringPalmVibrations(character.QuiveringPalmTarget{}, 5)
	wantInvalid(t, err, "zero target")
	_, err = character.NewQuiveringPalmVibrations(mnkTarget(), 0)
	wantInvalid(t, err, "zero days")
	if v, err := character.NewQuiveringPalmVibrations(mnkTarget(), 1); err != nil || !v.Active() || v.DurationDays() != 1 {
		t.Errorf("one day = %v, %v, want accepted and active", v, err)
	}
}

// Requirement: doc/specs/class-monk.md FR-144 - MonkState implements FeatureState: Equal is by value and false for another type of state; it is
// stored under the one key named monk and read back equal.
// Path: happy.
func TestMonkStateEqualityAndKey(t *testing.T) {
	t.Parallel()
	a := character.MustMonkState(1, true, 1, mnkVibrations(5))
	if !a.Equal(character.MustMonkState(1, true, 1, mnkVibrations(5))) || a.Equal(character.MustMonkState(2, true, 1, mnkVibrations(5))) ||
		a.Equal(character.MustMonkState(1, false, 1, mnkVibrations(5))) || a.Equal(character.MustMonkState(1, true, 2, mnkVibrations(5))) ||
		a.Equal(character.MustMonkState(1, true, 1, mnkVibrations(6))) || a.Equal(character.NewWeaponMasteryChoiceState(character.WeaponMasteryChoice{})) {
		t.Error("Equal must compare by value and reject another type")
	}
	if character.MonkStateKey().Name() != "monk" {
		t.Errorf("key = %q, want monk", character.MonkStateKey().Name())
	}
	c := mnkWith(t, mnkMonk(t, 3), a)
	if got := mnkStateOf(t, c); got != a {
		t.Errorf("stored state = %#v, want %#v", got, a)
	}
}

// Requirement: doc/specs/class-monk.md Table 4.4a, FR-033, FR-034, FR-035, FR-133, FR-136 - the Focus Points view: spend one of several, spend
// exactly all, one short, a cost above the available (the view unchanged), a zero or negative cost is an error; at level 1 the maximum is 0; and a
// higher level keeps the expended points (1 of 2 spent at level 2 is 1 of 3 spent at level 3, 2 available).
// Path: per case (rows of Table 4.4a).
func TestFocusPointsSpend(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name                        string
		level, expended, cost       int
		spent                       bool
		wantAvailable, wantExpended int
		invalid                     bool
	}{
		{"spend one of several", 2, 0, 1, true, 1, 1, false},
		{"spend exactly all", 2, 1, 1, true, 0, 2, false},
		{"one short", 2, 2, 1, false, 0, 2, false},
		{"cost above available", 2, 0, 3, false, 2, 0, false},
		{"zero cost", 2, 0, 0, false, 2, 0, true},
		{"negative cost", 2, 0, -1, false, 2, 0, true},
		{"level 1 has no points", 1, 0, 1, false, 0, 0, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := character.MustMonkState(c.expended, false, 0, character.QuiveringPalmVibrations{}).FocusPoints(mnkStats(t, c.level))
			got, spent, err := f.Spend(c.cost)
			if c.invalid {
				wantInvalid(t, err, c.name)
			} else if err != nil {
				t.Fatalf("Spend = %v", err)
			}
			if spent != c.spent || got.Available() != c.wantAvailable || got.Expended() != c.wantExpended {
				t.Errorf("Spend(%d) = %+v spent %v, want %d available %d expended spent %v", c.cost, got, spent, c.wantAvailable, c.wantExpended, c.spent)
			}
		})
	}
	one := character.MustMonkState(1, false, 0, character.QuiveringPalmVibrations{})
	if f := one.FocusPoints(mnkStats(t, 3)); f.Maximum() != 3 || f.Available() != 2 || f.Expended() != 1 {
		t.Errorf("1 of 2 expended at level 3 = %+v, want 2 available of 3", f)
	}
	if f := character.MustMonkState(9, false, 0, character.QuiveringPalmVibrations{}).FocusPoints(mnkStats(t, 2)); f.Available() != 0 {
		t.Errorf("more expended than the maximum = %d available, want 0", f.Available())
	}
}

// Requirement: doc/specs/class-monk.md Table 4.10b, FR-111, FR-134 - Wholeness of Body has the Wisdom modifier in uses, at least one; the
// remaining uses never go below 0 when the Wisdom modifier falls, and a later rise does not restore uses.
// Path: per case (rows of Table 4.10b).
func TestWholenessOfBodyUses(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ wis, uses int }{{-2, 1}, {0, 1}, {1, 1}, {3, 3}, {5, 5}} {
		if got := character.WholenessOfBodyUses(modifierOf(t, c.wis)); got != c.uses {
			t.Errorf("uses at Wisdom %+d = %d, want %d", c.wis, got, c.uses)
		}
	}
	state := mnkState(0, false, 3)
	for wis, want := range map[int]int{5: 2, 3: 0, 1: 0, -2: 0} {
		if got := state.WholenessOfBodyRemaining(modifierOf(t, wis)); got != want {
			t.Errorf("3 spent at Wisdom %+d = %d remaining, want %d", wis, got, want)
		}
	}
}

// Requirement: doc/specs/class-monk.md FR-036, FR-049, FR-124 - a Short Rest regains every expended Focus Point only; a Long Rest also restores
// Uncanny Metabolism and every Wholeness of Body use; neither ends active vibrations; any other kind changes nothing.
// Path: per case.
func TestMonkStateAfterRest(t *testing.T) {
	t.Parallel()
	full := character.MustMonkState(4, true, 2, mnkVibrations(9))
	for _, c := range []struct {
		name string
		kind character.RestKind
		want character.MonkState
	}{
		{"short rest", character.ShortRest, character.MustMonkState(0, true, 2, mnkVibrations(9))},
		{"long rest", character.LongRest, character.MustMonkState(0, false, 0, mnkVibrations(9))},
		{"an invalid kind", character.RestKind(0), full},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := full.AfterRest(c.kind); got != c.want {
				t.Errorf("AfterRest = %#v, want %#v", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-143, FR-150 - MonkCreationChoices seeds exactly the initial MonkState under the key monk, leaves the
// other keys and its argument unchanged, names no content (an empty non-nil slice) and satisfies ClassCreationChoices; a foreign state under the
// key is left as it is.
// Path: happy / unhappy (invalid input).
func TestMonkCreationChoices(t *testing.T) {
	t.Parallel()
	other := must(character.SetFeatureState(character.FeatureStates{}, character.MustFeatureStateKey[character.ChosenSubclass]("subclass_wizard"), must(character.NewChosenSubclass("evoker"))))
	var choices character.ClassCreationChoices = character.MonkCreationChoices{}
	seeded := choices.SeedFeatureStates(other)
	got, ok, err := character.ReadFeatureState(seeded, character.MonkStateKey())
	if err != nil || !ok || got != character.InitialMonkState() || seeded.Len() != 2 || other.Len() != 1 {
		t.Errorf("seeded = %v (len %d, original %d), state %v %v %v", seeded, seeded.Len(), other.Len(), got, ok, err)
	}
	if keys := choices.ContentKeys(); keys == nil || len(keys) != 0 {
		t.Errorf("ContentKeys = %#v, want an empty non-nil slice", keys)
	}
	foreign := must(character.SetFeatureState(character.FeatureStates{}, character.MustFeatureStateKey[character.ChosenSubclass]("monk"), must(character.NewChosenSubclass("warrior_of_the_open_hand"))))
	if got := choices.SeedFeatureStates(foreign); !got.Equal(foreign) {
		t.Errorf("a foreign state under monk was replaced: %v", got)
	}
}
