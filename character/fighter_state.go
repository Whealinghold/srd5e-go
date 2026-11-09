package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

// FighterState is the Fighter's class feature state: how many uses of Second
// Wind, Action Surge and Indomitable are expended, and the markers of the level 1
// Fighting Style and of the Champion's Additional Fighting Style (the feats
// themselves live in TakenFeats). It holds no level and no subclass, and no
// Weapon Mastery choice, which is kept under WeaponMasteryChoiceKey. It is
// stored under FighterStateKey; an absent state is Initial. It is immutable,
// comparable with ==, and the zero value is the Initial state. Safe for
// concurrent use.
type FighterState struct {
	secondWindExpended, actionSurgeExpended, indomitableExpended int
	fightingStyle, additionalFightingStyle                       content.FeatID
}

// FighterStateKey returns the FeatureStateKey named fighter under which a
// character holds its FighterState.
func FighterStateKey() FeatureStateKey[FighterState] {
	return MustFeatureStateKey[FighterState]("fighter")
}

// InitialFighterState returns the empty state: nothing expended, no Fighting
// Style marker.
func InitialFighterState() FighterState { return FighterState{} }

// NewFighterState returns the state with the given expended counts and markers
// (an empty id is no marker). It enforces only level-free invariants: it returns
// an error matching srderr.ErrInvalidInput for a negative count or a non-empty
// invalid feat id. Bounds that depend on the level are the domain functions'. It
// never panics.
func NewFighterState(secondWind, actionSurge, indomitable int, fightingStyle, additionalFightingStyle content.FeatID) (FighterState, error) {
	for _, n := range []int{secondWind, actionSurge, indomitable} {
		if n < 0 {
			return FighterState{}, invalidInput("Fighter state with a negative expended count " + strconv.Itoa(n))
		}
	}
	for _, f := range []content.FeatID{fightingStyle, additionalFightingStyle} {
		if f != "" && !f.Valid() {
			return FighterState{}, invalidInput("Fighter state with an invalid Fighting Style feat id")
		}
	}
	return FighterState{secondWindExpended: secondWind, actionSurgeExpended: actionSurge, indomitableExpended: indomitable,
		fightingStyle: fightingStyle, additionalFightingStyle: additionalFightingStyle}, nil
}

// MustFighterState is NewFighterState for arguments known to be valid. It panics
// with the error NewFighterState returns, and only then.
func MustFighterState(secondWind, actionSurge, indomitable int, fightingStyle, additionalFightingStyle content.FeatID) FighterState {
	s, err := NewFighterState(secondWind, actionSurge, indomitable, fightingStyle, additionalFightingStyle)
	if err != nil {
		panic(err)
	}
	return s
}

// SecondWindExpended returns the expended uses of Second Wind.
func (s FighterState) SecondWindExpended() int { return s.secondWindExpended }

// ActionSurgeExpended returns the expended uses of Action Surge.
func (s FighterState) ActionSurgeExpended() int { return s.actionSurgeExpended }

// IndomitableExpended returns the expended uses of Indomitable.
func (s FighterState) IndomitableExpended() int { return s.indomitableExpended }

// FightingStyle returns the marker of the level 1 Fighting Style, or "" for none.
func (s FighterState) FightingStyle() content.FeatID { return s.fightingStyle }

// AdditionalFightingStyle returns the marker of the Champion's Additional
// Fighting Style, or "" for none.
func (s FighterState) AdditionalFightingStyle() content.FeatID { return s.additionalFightingStyle }

// Equal reports whether other is a FighterState with the same counts and markers.
func (s FighterState) Equal(other FeatureState) bool {
	o, ok := other.(FighterState)
	return ok && o == s
}

// remaining returns the uses left of a maximum with expended spent, never below 0.
func remaining(maximum, expended int) int { return max(maximum-expended, 0) }

// SecondWindRemaining returns the unexpended uses of Second Wind at stats' level:
// the maximum less the expended count, never below 0.
func (s FighterState) SecondWindRemaining(stats FighterLevelStats) int {
	return remaining(stats.secondWind, s.secondWindExpended)
}

// ActionSurgeRemaining returns the unexpended uses of Action Surge at stats' level.
func (s FighterState) ActionSurgeRemaining(stats FighterLevelStats) int {
	return remaining(stats.actionSurge, s.actionSurgeExpended)
}

// IndomitableRemaining returns the unexpended uses of Indomitable at stats' level.
func (s FighterState) IndomitableRemaining(stats FighterLevelStats) int {
	return remaining(stats.indomitable, s.indomitableExpended)
}

// AfterRest returns the state after the benefits of a rest of kind: a Short Rest
// regains one expended Second Wind use and every Action Surge use, a Long Rest
// regains every use of all three (SRD 5.2.1, "Level 1: Second Wind", "Level 2:
// Action Surge", "Level 9: Indomitable", p. 48). Any other kind changes nothing.
// The host applies it when a RestApplied for this character arrives, passing
// RestApplied.BenefitsOf.
func (s FighterState) AfterRest(kind RestKind) FighterState {
	switch kind {
	case ShortRest:
		s.secondWindExpended = max(s.secondWindExpended-1, 0)
		s.actionSurgeExpended = 0
	case LongRest:
		s.secondWindExpended, s.actionSurgeExpended, s.indomitableExpended = 0, 0, 0
	}
	return s
}

// FighterCreationChoices is the ClassCreationChoices of a new Fighter: it seeds
// the initial FighterState under FighterStateKey and names no content (the skills
// and the Starting Equipment letter belong to character creation). It has no
// fields; the zero value is ready to use. Safe for concurrent use.
type FighterCreationChoices struct{}

// SeedFeatureStates returns states with the initial FighterState stored under
// FighterStateKey and every other entry unchanged; states itself is unchanged.
func (FighterCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	next, err := SetFeatureState(states, FighterStateKey(), InitialFighterState())
	if err != nil { // the key is valid, the state is non-nil; only a foreign type under "fighter" fails
		return states
	}
	return next
}

// ContentKeys returns an empty slice: the Fighter choices name no content.
func (FighterCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

var _ ClassCreationChoices = FighterCreationChoices{}
