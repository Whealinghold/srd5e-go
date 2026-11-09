package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// QuiveringPalmTarget is the opaque reference to the creature under Quivering Palm
// vibrations. The host supplies it; the character context cannot hold a combatant
// id and does not read it. Build it with NewQuiveringPalmTarget; the zero value
// refers to no creature. Comparable with ==, safe for concurrent use.
type QuiveringPalmTarget struct{ ref string }

// NewQuiveringPalmTarget returns the target reference ref. It returns an error
// matching srderr.ErrInvalidInput for an empty reference or one over 128 bytes. It
// never panics.
func NewQuiveringPalmTarget(ref string) (QuiveringPalmTarget, error) {
	if ref == "" || len(ref) > 128 {
		return QuiveringPalmTarget{}, invalidInput("Quivering Palm target reference must be 1 to 128 bytes")
	}
	return QuiveringPalmTarget{ref: ref}, nil
}

// Ref returns the reference.
func (t QuiveringPalmTarget) Ref() string { return t.ref }

// IsZero reports whether t refers to no creature.
func (t QuiveringPalmTarget) IsZero() bool { return t.ref == "" }

// QuiveringPalmVibrations are the vibrations a Monk set up in one creature: the
// target and the days they last, counted from the Monk level when they started
// (SRD 5.2.1, "Level 17: Quivering Palm", p. 52). The engine never reads a clock.
// The zero value is no vibrations. Comparable with ==.
type QuiveringPalmVibrations struct {
	target QuiveringPalmTarget
	days   int
}

// NewQuiveringPalmVibrations returns vibrations on target lasting days days. It
// returns an error matching srderr.ErrInvalidInput for a zero target or days below 1.
func NewQuiveringPalmVibrations(target QuiveringPalmTarget, days int) (QuiveringPalmVibrations, error) {
	if target.IsZero() || days < 1 {
		return QuiveringPalmVibrations{}, invalidInput("Quivering Palm vibrations need a target and at least 1 day")
	}
	return QuiveringPalmVibrations{target: target, days: days}, nil
}

// Target returns the creature the vibrations are in.
func (v QuiveringPalmVibrations) Target() QuiveringPalmTarget { return v.target }

// DurationDays returns the days the vibrations last.
func (v QuiveringPalmVibrations) DurationDays() int { return v.days }

// Active reports whether vibrations are set up.
func (v QuiveringPalmVibrations) Active() bool { return v.days > 0 }

// MonkState is the Monk's class feature state: the expended Focus Points, whether
// Uncanny Metabolism was used since the last Long Rest, the spent uses of
// Wholeness of Body and the Quivering Palm vibrations. It holds no level, no
// maximum and no subclass (the maxima are derived on read, class-monk.md FR-133).
// It is stored under MonkStateKey; an absent state is Initial. It is immutable,
// comparable with ==, and the zero value is the Initial state. Safe for
// concurrent use.
type MonkState struct {
	focusExpended  int
	uncannyUsed    bool
	wholenessSpent int
	vibrations     QuiveringPalmVibrations
}

// MonkStateKey returns the FeatureStateKey named monk under which a character
// holds its MonkState.
func MonkStateKey() FeatureStateKey[MonkState] { return MustFeatureStateKey[MonkState]("monk") }

// InitialMonkState returns the empty state: nothing expended, no vibrations.
func InitialMonkState() MonkState { return MonkState{} }

// NewMonkState returns the state with the given expended counts. It enforces only
// level-free invariants: it returns an error matching srderr.ErrInvalidInput for a
// negative count. Bounds that depend on the level are the domain functions'. It
// never panics.
func NewMonkState(focusExpended int, uncannyUsed bool, wholenessSpent int, vibrations QuiveringPalmVibrations) (MonkState, error) {
	if focusExpended < 0 || wholenessSpent < 0 {
		return MonkState{}, invalidInput("Monk state with a negative expended count " + strconv.Itoa(min(focusExpended, wholenessSpent)))
	}
	return MonkState{focusExpended: focusExpended, uncannyUsed: uncannyUsed, wholenessSpent: wholenessSpent, vibrations: vibrations}, nil
}

// MustMonkState is NewMonkState for arguments known to be valid. It panics with the
// error NewMonkState returns, and only then.
func MustMonkState(focusExpended int, uncannyUsed bool, wholenessSpent int, vibrations QuiveringPalmVibrations) MonkState {
	s, err := NewMonkState(focusExpended, uncannyUsed, wholenessSpent, vibrations)
	if err != nil {
		panic(err)
	}
	return s
}

// FocusExpended returns the expended Focus Points.
func (s MonkState) FocusExpended() int { return s.focusExpended }

// UncannyMetabolismUsed reports whether Uncanny Metabolism was used since the last Long Rest.
func (s MonkState) UncannyMetabolismUsed() bool { return s.uncannyUsed }

// WholenessOfBodySpent returns the spent uses of Wholeness of Body.
func (s MonkState) WholenessOfBodySpent() int { return s.wholenessSpent }

// Vibrations returns the Quivering Palm vibrations; the zero value when none.
func (s MonkState) Vibrations() QuiveringPalmVibrations { return s.vibrations }

// Equal reports whether other is a MonkState with the same counts and vibrations.
func (s MonkState) Equal(other FeatureState) bool {
	o, ok := other.(MonkState)
	return ok && o == s
}

// FocusPoints is the read-time view of a Monk's Focus Points: the maximum of the
// level and the expended count (SRD 5.2.1, "Level 2: Monk's Focus", p. 50). The
// expended count may exceed a lowered maximum; Available is then 0. Immutable,
// comparable with ==.
type FocusPoints struct{ maximum, expended int }

// Maximum returns the Focus Points maximum.
func (f FocusPoints) Maximum() int { return f.maximum }

// Expended returns the expended Focus Points.
func (f FocusPoints) Expended() int { return f.expended }

// Available returns the unexpended Focus Points, never below 0.
func (f FocusPoints) Available() int { return remaining(f.maximum, f.expended) }

// Spend expends cost Focus Points. When fewer are available it returns the same
// value and false, a rule outcome and not an error. It returns an error matching
// srderr.ErrInvalidInput for a cost below 1, a caller bug. It never panics.
func (f FocusPoints) Spend(cost int) (FocusPoints, bool, error) {
	if cost < 1 {
		return f, false, invalidInput("Focus Point cost " + strconv.Itoa(cost) + " below 1")
	}
	if cost > f.Available() {
		return f, false, nil
	}
	f.expended += cost
	return f, true, nil
}

// FocusPoints returns the view of the Focus Points at stats' level.
func (s MonkState) FocusPoints(stats MonkLevelStats) FocusPoints {
	return FocusPoints{maximum: stats.focusPoints, expended: s.focusExpended}
}

// WholenessOfBodyUses returns the maximum uses of Wholeness of Body: the Wisdom
// modifier, minimum once (SRD 5.2.1, "Level 6: Wholeness of Body", p. 52).
func WholenessOfBodyUses(wisdom ability.Modifier) int { return max(wisdom.Value(), 1) }

// WholenessOfBodyRemaining returns the unspent uses of Wholeness of Body for the
// given Wisdom modifier, never below 0 (class-monk.md FR-134).
func (s MonkState) WholenessOfBodyRemaining(wisdom ability.Modifier) int {
	return remaining(WholenessOfBodyUses(wisdom), s.wholenessSpent)
}

// AfterRest returns the state after the benefits of a rest of kind: a Short Rest
// regains every expended Focus Point; a Long Rest also restores Uncanny Metabolism
// and every use of Wholeness of Body (SRD 5.2.1, "Level 2: Monk's Focus", "Level 2:
// Uncanny Metabolism", p. 50-51, "Level 6: Wholeness of Body", p. 52). Quivering
// Palm vibrations last days and no rest ends them (class-monk.md FR-124). Any other
// kind changes nothing. The host applies it when a RestApplied for this character
// arrives, passing RestApplied.BenefitsOf.
func (s MonkState) AfterRest(kind RestKind) MonkState {
	switch kind {
	case ShortRest:
		s.focusExpended = 0
	case LongRest:
		s.focusExpended, s.uncannyUsed, s.wholenessSpent = 0, false, 0
	}
	return s
}

// MonkCreationChoices is the ClassCreationChoices of a new Monk: it seeds the
// initial MonkState under MonkStateKey and names no content (the skills, the tool
// and the Starting Equipment letter belong to character creation). It has no
// fields; the zero value is ready to use. Safe for concurrent use.
type MonkCreationChoices struct{}

// SeedFeatureStates returns states with the initial MonkState stored under
// MonkStateKey and every other entry unchanged; states itself is unchanged.
func (MonkCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	next, err := SetFeatureState(states, MonkStateKey(), InitialMonkState())
	if err != nil { // the key is valid, the state is non-nil; only a foreign type under "monk" fails
		return states
	}
	return next
}

// ContentKeys returns an empty slice: the Monk choices name no content.
func (MonkCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

var _ ClassCreationChoices = MonkCreationChoices{}
