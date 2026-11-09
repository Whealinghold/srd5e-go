package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// StrokeOfLuckState says whether Stroke of Luck can be used. The zero value is StrokeOfLuckStateAvailable, so an absent
// state is the initial one.
type StrokeOfLuckState int

const (
	// StrokeOfLuckStateAvailable: the feature has not been used since the last Short or Long Rest.
	StrokeOfLuckStateAvailable StrokeOfLuckState = iota
	// StrokeOfLuckStateSpent: the feature was used and no rest has finished since.
	StrokeOfLuckStateSpent
)

// String returns "available", "spent" or "unknown".
func (s StrokeOfLuckState) String() string {
	switch s {
	case StrokeOfLuckStateAvailable:
		return "available"
	case StrokeOfLuckStateSpent:
		return "spent"
	}
	return "unknown"
}

// Valid reports whether s is one of the two states.
func (s StrokeOfLuckState) Valid() bool {
	return s == StrokeOfLuckStateAvailable || s == StrokeOfLuckStateSpent
}

// RogueState is the Rogue's class feature state: whether Stroke of Luck is spent, which Expertise windows were used and the
// language chosen with Thieves' Cant. It holds no level and no subclass (the subclass is recorded by the subclass choice) and
// no Weapon Mastery choice, which is kept under WeaponMasteryChoiceKey. It is stored under RogueStateKey; an absent state is
// the initial one. It is immutable, comparable with ==, and the zero value is the initial state. Safe for concurrent use.
type RogueState struct {
	stroke      StrokeOfLuckState
	levelOne    bool
	levelSix    bool
	otherTongue content.Language
}

// RogueStateKey returns the FeatureStateKey named rogue under which a character holds its RogueState.
func RogueStateKey() FeatureStateKey[RogueState] { return MustFeatureStateKey[RogueState]("rogue") }

// InitialRogueState returns the empty state: Stroke of Luck available, no Expertise window used, no language recorded.
func InitialRogueState() RogueState { return RogueState{} }

// NewRogueState returns the state with the given Stroke of Luck state, used Expertise windows and language (the zero Language
// is none). It returns an error matching srderr.ErrInvalidInput for an undefined Stroke of Luck state or window, or a nonzero
// language that is not valid. It never panics.
func NewRogueState(stroke StrokeOfLuckState, windows []RogueExpertiseWindow, language content.Language) (RogueState, error) {
	if !stroke.Valid() || (language != 0 && !language.Valid()) {
		return RogueState{}, invalidInput("Rogue state with an undefined Stroke of Luck state or language")
	}
	s := RogueState{stroke: stroke, otherTongue: language}
	for _, w := range windows {
		switch w {
		case ExpertiseLevelOne:
			s.levelOne = true
		case ExpertiseLevelSix:
			s.levelSix = true
		default:
			return RogueState{}, invalidInput("Rogue state with an undefined Expertise window")
		}
	}
	return s, nil
}

// StrokeOfLuck returns the Stroke of Luck state.
func (s RogueState) StrokeOfLuck() StrokeOfLuckState { return s.stroke }

// ExpertiseWindowsUsed returns the Expertise windows already used as a new slice, level 1 first.
func (s RogueState) ExpertiseWindowsUsed() []RogueExpertiseWindow {
	out := []RogueExpertiseWindow{}
	if s.levelOne {
		out = append(out, ExpertiseLevelOne)
	}
	if s.levelSix {
		out = append(out, ExpertiseLevelSix)
	}
	return out
}

// OtherLanguage returns the language chosen with Thieves' Cant, or the zero Language for none.
func (s RogueState) OtherLanguage() content.Language { return s.otherTongue }

// WithExpertise returns the state with the window of an accepted selection marked used; s is unchanged. An undefined window
// changes nothing.
func (s RogueState) WithExpertise(sel RogueExpertiseSelection) RogueState {
	switch sel.window {
	case ExpertiseLevelOne:
		s.levelOne = true
	case ExpertiseLevelSix:
		s.levelSix = true
	}
	return s
}

// Equal reports whether other is a RogueState with the same fields.
func (s RogueState) Equal(other FeatureState) bool {
	o, ok := other.(RogueState)
	return ok && o == s
}

// AfterRest returns the state after the benefits of a rest of kind: a Short Rest or a Long Rest makes a spent Stroke of Luck
// available again (SRD 5.2.1, "Level 20: Stroke of Luck", p. 63). Any other kind changes nothing. The host applies it when a
// RestApplied for this character arrives, passing RestApplied.BenefitsOf.
func (s RogueState) AfterRest(kind RestKind) RogueState {
	if kind == ShortRest || kind == LongRest {
		s.stroke = StrokeOfLuckStateAvailable
	}
	return s
}

var _ FeatureState = RogueState{}
