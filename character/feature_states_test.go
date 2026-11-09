package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Two test-only feature states of different types, standing in for what class
// specs will store.
type rageState struct{ uses int }

func (s rageState) Equal(o character.FeatureState) bool { r, ok := o.(rageState); return ok && r == s }

type surgeState struct{ used bool }

func (s surgeState) Equal(o character.FeatureState) bool {
	r, ok := o.(surgeState)
	return ok && r == s
}

// Requirement: doc/specs/character-aggregate.md FR-093 - a key name must match
// [a-z][a-z0-9_]*; same-named keys are equal.
// Path: per case.
func TestFeatureStateKeyNames(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		ok   bool
	}{{"rage", true}, {"a1_b2", true}, {"", false}, {"Rage", false}, {"1rage", false}, {"_rage", false}, {"ra-ge", false}, {"ra ge", false}, {"rage9", true}} {
		k, err := character.NewFeatureStateKey[rageState](c.name)
		if (err == nil) != c.ok {
			t.Errorf("%q: err %v", c.name, err)
		}
		if !c.ok {
			wantInvalid(t, err, c.name)
			if recovered(t, func() { character.MustFeatureStateKey[rageState](c.name) }) == nil {
				t.Errorf("%q: Must must panic", c.name)
			}
		} else if k.Name() != c.name {
			t.Errorf("Name = %q", k.Name())
		}
	}
	first, second := character.MustFeatureStateKey[rageState]("rage"), character.MustFeatureStateKey[rageState]("rage")
	if first != second {
		t.Error("same-named keys must be equal")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-092, FR-093 - read absent,
// write, overwrite, a second foreign key, a mismatched type is a validation
// error, and the aggregate keeps states by value without interpreting them.
// Path: per case.
func TestFeatureStates(t *testing.T) {
	t.Parallel()
	rage := character.MustFeatureStateKey[rageState]("rage")
	surge := character.MustFeatureStateKey[surgeState]("surge")
	var empty character.FeatureStates
	if _, ok, err := character.ReadFeatureState(empty, rage); ok || err != nil || empty.Len() != 0 {
		t.Errorf("absent read: %v %v", ok, err)
	}
	s1, err := character.SetFeatureState(empty, rage, rageState{2})
	if err != nil || empty.Len() != 0 {
		t.Fatalf("set: %v, original len %d", err, empty.Len())
	}
	s2, err := character.SetFeatureState(s1, surge, surgeState{true})
	if err != nil {
		t.Fatal(err)
	}
	s3, err := character.SetFeatureState(s2, rage, rageState{1}) // overwrite
	if err != nil {
		t.Fatal(err)
	}
	if v, ok, _ := character.ReadFeatureState(s2, rage); !ok || v.uses != 2 {
		t.Errorf("s2 rage = %v", v)
	}
	if v, ok, _ := character.ReadFeatureState(s3, rage); !ok || v.uses != 1 {
		t.Errorf("s3 rage = %v", v)
	}
	if v, ok, _ := character.ReadFeatureState(s3, surge); !ok || !v.used {
		t.Errorf("s3 surge = %v", v)
	}
	if s2.Equal(s3) || !s2.Equal(s2) || s1.Equal(s2) || s3.Len() != 2 {
		t.Error("Equal compares names and states by value")
	}
	// Same name, other type: both read and write are validation errors.
	wrong := character.MustFeatureStateKey[surgeState]("rage")
	_, _, err = character.ReadFeatureState(s1, wrong)
	wantInvalid(t, err, "read with a mismatched type")
	_, err = character.SetFeatureState(s1, wrong, surgeState{})
	wantInvalid(t, err, "write with a mismatched type")
	// Zero key and nil state.
	_, _, err = character.ReadFeatureState(s1, character.FeatureStateKey[rageState]{})
	wantInvalid(t, err, "zero key read")
	_, err = character.SetFeatureState(s1, character.FeatureStateKey[rageState]{}, rageState{})
	wantInvalid(t, err, "zero key write")
	_, err = character.SetFeatureState[character.FeatureState](s1, character.MustFeatureStateKey[character.FeatureState]("any"), nil)
	wantInvalid(t, err, "nil state")
}

// Requirement: doc/specs/character-aggregate.md FR-092, FR-097 - WithFeatureState
// returns a new Character holding the state and every other entry unchanged, with
// the input untouched; a zero Character is rejected.
// Path: happy.
func TestWithFeatureState(t *testing.T) {
	t.Parallel()
	rage := character.MustFeatureStateKey[rageState]("rage")
	c := created(t, "fighter", flatScores(t, 10))
	next, err := character.WithFeatureState(c, rage, rageState{3})
	if err != nil {
		t.Fatal(err)
	}
	if c.FeatureStates().Len() != 0 || next.FeatureStates().Len() != 1 || next.Equal(c) {
		t.Error("the input must be unchanged and the result must hold the state")
	}
	if v, _, _ := character.ReadFeatureState(next.FeatureStates(), rage); v.uses != 3 {
		t.Errorf("stored = %v", v)
	}
	if _, err := character.WithFeatureState(character.Character{}, rage, rageState{}); !isInvalid(err) {
		t.Errorf("zero character: %v", err)
	}
	if _, err := character.WithFeatureState(c, character.FeatureStateKey[rageState]{}, rageState{}); !isInvalid(err) {
		t.Errorf("zero key: %v", err)
	}
}

func isInvalid(err error) bool { return err != nil && errors.Is(err, srderr.ErrInvalidInput) }
