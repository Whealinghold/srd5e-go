package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

// Levels at which a Barbarian feature is gained (SRD 5.2.1, "Barbarian Features", p. 28, and the
// feature headings pp. 29-30). They are rule constants; the progression test checks them against the
// printed feature levels of the bundled definition.
const (
	dangerSenseLevel         = 2
	recklessAttackLevel      = 2
	primalKnowledgeLevel     = 3
	frenzySubclassLevel      = 3
	extraAttackLevel         = 5
	mindlessRageLevel        = 6
	feralInstinctLevel       = 7
	brutalStrikeLevel        = 9
	retaliationLevel         = 10
	relentlessRageLevel      = 11
	improvedBrutalStrike13   = 13
	intimidatingPresenceLvl  = 14
	persistentRageLevel      = 15
	improvedBrutalStrike17   = 17
	indomitableMightLevel    = 18
	primalChampionLevel      = 20
	relentlessRageStartingDC = 10
	relentlessRageDCStep     = 5
)

// BarbarianState is the Barbarian's class feature state: how many Rage uses are
// expended, whether a Rage is active and how it stands this turn, how many times
// Relentless Rage was used since the last rest, and whether Persistent Rage and
// Intimidating Presence are spent. It holds no level, no subclass, no Weapon Mastery
// choice (kept under WeaponMasteryChoiceKey) and no per-turn Bonus Action or
// Reaction (BarbarianTurnState). It is stored under BarbarianStateKey; an absent
// state is Initial. It is immutable, comparable with ==, and the zero value is the
// Initial state. Safe for concurrent use.
type BarbarianState struct {
	ragesExpended, relentlessRageUses int
	rageActive, rageBeganThisTurn     bool
	rageExtendedThisTurn              bool
	persistentRageUsed                bool
	presenceUsed                      bool
}

// BarbarianStateKey returns the FeatureStateKey named barbarian under which a
// character holds its BarbarianState.
func BarbarianStateKey() FeatureStateKey[BarbarianState] {
	return MustFeatureStateKey[BarbarianState]("barbarian")
}

// InitialBarbarianState returns the empty state: nothing expended, no Rage active.
func InitialBarbarianState() BarbarianState { return BarbarianState{} }

// NewBarbarianState returns the state with ragesExpended Rage uses expended,
// relentlessRageUses uses of Relentless Rage since the last rest and no Rage active. It
// returns an error matching srderr.ErrInvalidInput for a negative count. Bounds that
// depend on the level are the domain functions'. It never panics.
func NewBarbarianState(ragesExpended, relentlessRageUses int) (BarbarianState, error) {
	if ragesExpended < 0 || relentlessRageUses < 0 {
		return BarbarianState{}, invalidInput("Barbarian state with a negative count " + strconv.Itoa(min(ragesExpended, relentlessRageUses)))
	}
	return BarbarianState{ragesExpended: ragesExpended, relentlessRageUses: relentlessRageUses}, nil
}

// MustBarbarianState is NewBarbarianState for arguments known to be valid. It panics
// with the error NewBarbarianState returns, and only then.
func MustBarbarianState(ragesExpended, relentlessRageUses int) BarbarianState {
	s, err := NewBarbarianState(ragesExpended, relentlessRageUses)
	if err != nil {
		panic(err)
	}
	return s
}

// RagesExpended returns the expended Rage uses.
func (s BarbarianState) RagesExpended() int { return s.ragesExpended }

// RelentlessRageUses returns how many times Relentless Rage was used since the last rest.
func (s BarbarianState) RelentlessRageUses() int { return s.relentlessRageUses }

// RageActive reports whether a Rage is active.
func (s BarbarianState) RageActive() bool { return s.rageActive }

// PersistentRageUsed reports whether Persistent Rage was used since the last Long Rest.
func (s BarbarianState) PersistentRageUsed() bool { return s.persistentRageUsed }

// IntimidatingPresenceUsed reports whether Intimidating Presence is spent: used since
// the last Long Rest and not restored.
func (s BarbarianState) IntimidatingPresenceUsed() bool { return s.presenceUsed }

// RelentlessRageDC returns the Difficulty Class of the next Relentless Rage: 10, and
// 5 more for each use since the last rest (SRD 5.2.1, "Level 11: Relentless Rage", p. 30).
func (s BarbarianState) RelentlessRageDC() int {
	return relentlessRageStartingDC + relentlessRageDCStep*s.relentlessRageUses
}

// Equal reports whether other is a BarbarianState with the same fields.
func (s BarbarianState) Equal(other FeatureState) bool {
	o, ok := other.(BarbarianState)
	return ok && o == s
}

// RagesRemaining returns the unexpended Rage uses at stats' level: the maximum less
// the expended count, never below 0.
func (s BarbarianState) RagesRemaining(stats BarbarianLevelStats) int {
	return remaining(stats.rages, s.ragesExpended)
}

// AfterRest returns the state after the benefits of a rest of kind: a Short Rest
// regains one expended Rage use and resets the Relentless Rage DC; a Long Rest regains
// every Rage use, resets the DC and makes Persistent Rage and Intimidating Presence
// available again (SRD 5.2.1, "Level 1: Rage", p. 28, "Level 11: Relentless Rage",
// "Level 15: Persistent Rage", "Level 14: Intimidating Presence", p. 30). A rest also
// ends a Rage, which lasts 10 minutes at most. Any other kind changes nothing. The host
// applies it when a RestApplied for this character arrives, passing RestApplied.BenefitsOf.
func (s BarbarianState) AfterRest(kind RestKind) BarbarianState {
	switch kind {
	case ShortRest:
		s.ragesExpended = max(s.ragesExpended-1, 0)
	case LongRest:
		s.ragesExpended, s.persistentRageUsed, s.presenceUsed = 0, false, false
	default:
		return s
	}
	s.relentlessRageUses = 0
	s.rageActive, s.rageBeganThisTurn, s.rageExtendedThisTurn = false, false, false
	return s
}

// BarbarianCreationChoices is the ClassCreationChoices of a new Barbarian: it seeds
// the initial BarbarianState under BarbarianStateKey and names no content (the skills
// and the Starting Equipment letter belong to character creation, and the first Weapon
// Mastery kinds are chosen with ApplyBarbarianOptionChoice). It has no fields; the zero
// value is ready to use. Safe for concurrent use.
type BarbarianCreationChoices struct{}

// SeedFeatureStates returns states with the initial BarbarianState stored under
// BarbarianStateKey and every other entry unchanged; states itself is unchanged.
func (BarbarianCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	next, err := SetFeatureState(states, BarbarianStateKey(), InitialBarbarianState())
	if err != nil { // the key is valid, the state is non-nil; only a foreign type under "barbarian" fails
		return states
	}
	return next
}

// ContentKeys returns an empty slice: the Barbarian choices name no content.
func (BarbarianCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

var _ ClassCreationChoices = BarbarianCreationChoices{}
