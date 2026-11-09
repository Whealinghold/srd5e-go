package character

import (
	"maps"
)

// FeatureState is the state of one class or species feature, owned by the spec
// of that feature. The Character stores it and never interprets it. An
// implementation MUST compare by value and be immutable.
type FeatureState interface {
	// Equal reports whether other is the same state, by value.
	Equal(other FeatureState) bool
}

// FeatureStateKey names one feature state and fixes its type T, so reads and
// writes need no type assertion by the caller. Two keys with the same name and
// T are equal with ==. The zero key is invalid; build one with
// NewFeatureStateKey.
type FeatureStateKey[T FeatureState] struct{ name string }

// FeatureStates is the immutable set of feature states a Character holds, by key
// name; the zero value is the empty set. Compare two with Equal, not ==. Safe
// for concurrent use.
type FeatureStates struct{ m map[string]FeatureState }

// NewFeatureStateKey returns the key called name, which must match
// [a-z][a-z0-9_]*. It returns an error matching srderr.ErrInvalidInput
// otherwise. It never panics.
func NewFeatureStateKey[T FeatureState](name string) (FeatureStateKey[T], error) {
	if !validFeatureName(name) {
		return FeatureStateKey[T]{}, invalidInput("feature state key name")
	}
	return FeatureStateKey[T]{name: name}, nil
}

// MustFeatureStateKey is NewFeatureStateKey for constants known to be valid. It
// panics with the error NewFeatureStateKey returns, and only then.
func MustFeatureStateKey[T FeatureState](name string) FeatureStateKey[T] {
	k, err := NewFeatureStateKey[T](name)
	if err != nil {
		panic(err)
	}
	return k
}

// Name returns the key's name.
func (k FeatureStateKey[T]) Name() string { return k.name }

// Len returns how many states are held.
func (s FeatureStates) Len() int { return len(s.m) }

// Equal reports whether s and o hold the same names with equal states.
func (s FeatureStates) Equal(o FeatureStates) bool {
	return maps.EqualFunc(s.m, o.m, func(a, b FeatureState) bool { return a.Equal(b) })
}

// ReadFeatureState returns the state stored under key and true, or the zero T
// and false when none is stored. It returns an error matching
// srderr.ErrInvalidInput when key is invalid or the name holds a state of
// another type than T. It never panics.
func ReadFeatureState[T FeatureState](s FeatureStates, key FeatureStateKey[T]) (T, bool, error) {
	var zero T
	if key.name == "" {
		return zero, false, invalidInput("zero feature state key")
	}
	stored, ok := s.m[key.name]
	if !ok {
		return zero, false, nil
	}
	typed, ok := stored.(T)
	if !ok {
		return zero, false, invalidInput("feature state of another type under " + key.name)
	}
	return typed, true, nil
}

// SetFeatureState returns a new FeatureStates holding state under key and every
// other entry of s unchanged; s is unchanged. It returns an error matching
// srderr.ErrInvalidInput when key is invalid, state is nil, or the name already
// holds a state of another type than T. It never panics.
func SetFeatureState[T FeatureState](s FeatureStates, key FeatureStateKey[T], state T) (FeatureStates, error) {
	if key.name == "" {
		return FeatureStates{}, invalidInput("zero feature state key")
	}
	if any(state) == nil {
		return FeatureStates{}, invalidInput("nil feature state")
	}
	if _, _, err := ReadFeatureState(s, key); err != nil {
		return FeatureStates{}, err
	}
	next := make(map[string]FeatureState, len(s.m)+1)
	maps.Copy(next, s.m)
	next[key.name] = state
	return FeatureStates{m: next}, nil
}

// WithFeatureState returns c with state stored under key and everything else
// unchanged, and no events. The receiver is unchanged. It returns an error
// matching srderr.ErrInvalidInput for the reasons of SetFeatureState or a zero
// c.
func WithFeatureState[T FeatureState](c Character, key FeatureStateKey[T], state T) (Character, error) {
	if c.IsZero() {
		return Character{}, invalidInput("zero character")
	}
	next, err := SetFeatureState(c.states, key, state)
	if err != nil {
		return Character{}, err
	}
	c.states = next
	return c, nil
}

// validFeatureName reports whether name matches [a-z][a-z0-9_]*.
func validFeatureName(name string) bool {
	for i, r := range name {
		letter := r >= 'a' && r <= 'z'
		digit := r >= '0' && r <= '9' || r == '_'
		if !letter && (i == 0 || !digit) {
			return false
		}
	}
	return name != ""
}
