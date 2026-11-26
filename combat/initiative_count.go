package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// InitiativeCount is a combatant's Initiative: the total of its Initiative check, or
// its Initiative score (SRD 5.2.1, "Combat", "Initiative", p. 13). Any integer,
// negative and zero included. Comparable with ==.
type InitiativeCount int

// FromRoll returns the count of a rolled Initiative: the d20 used (roll.Natural())
// plus the Initiative modifier plus the roll-only modifier. A natural 20 or 1 has no
// special effect. The receiver is ignored; the method is the one place the sum is
// computed, so a host that rerolls or replaces a die recomputes with it. Sums saturate.
func (InitiativeCount) FromRoll(roll d20.Roll, initiative, rollOnly modifier.Modifier) InitiativeCount {
	return InitiativeCount(satAdd(satAdd(roll.Natural(), initiative.Value()), rollOnly.Value()))
}

// InitiativeMethod says how a participant's count is made. The zero value is invalid.
type InitiativeMethod int

// The two methods.
const (
	// InitiativeMethodRolled: roll a d20 and add the modifiers.
	InitiativeMethodRolled InitiativeMethod = iota + 1
	// InitiativeMethodScore: 10 plus the modifier, +5 with Advantage, -5 with Disadvantage; no die.
	InitiativeMethodScore
)

// Valid reports whether m is one of the two methods.
func (m InitiativeMethod) Valid() bool {
	return m == InitiativeMethodRolled || m == InitiativeMethodScore
}

// String returns "rolled", "score" or "unknown".
func (m InitiativeMethod) String() string { return enumName([]string{"rolled", "score"}, int(m)) }

// Surprise says whether a participant is surprised, which gives Disadvantage on its
// Initiative roll (SRD 5.2.1, "Surprise", p. 189). The zero value is NotSurprised.
type Surprise int

// The two values.
const (
	NotSurprised Surprise = iota
	Surprised
)

// InitiativeParticipantInput holds the values NewInitiativeParticipant validates.
// Build it with a keyed literal.
type InitiativeParticipantInput struct {
	// IDs name the combatants rolled together (identical creatures); at least one.
	IDs []CombatantID
	// Modifier is the whole Initiative modifier (typically Dexterity).
	Modifier modifier.Modifier
	// RollOnly is added to a rolled count only, never to a score (Exhaustion).
	RollOnly modifier.Modifier
	Surprise Surprise
	// Contributions are the circumstances the conditions give the roll.
	Contributions RollContributions
	// Circumstances are other Advantage or Disadvantage sources.
	Circumstances []modifier.RollCircumstance
}

// InitiativeParticipant is one entry of RollInitiative: one or more combatants rolled
// once. Build it with NewInitiativeParticipant; the zero value is invalid. Immutable.
type InitiativeParticipant struct {
	ids           []CombatantID
	mod, rollOnly modifier.Modifier
	surprise      Surprise
	contributions RollContributions
	others        []modifier.RollCircumstance
}

// NewInitiativeParticipant returns the participant. It returns an error matching
// srderr.ErrInvalidInput for no id, a blank id, an id repeated within the entry or
// a Surprise that is not defined. The lists are copied. It never panics.
func NewInitiativeParticipant(in InitiativeParticipantInput) (InitiativeParticipant, error) {
	if len(in.IDs) == 0 || slices.ContainsFunc(in.IDs, CombatantID.IsZero) {
		return InitiativeParticipant{}, invalidInput("an Initiative entry names at least one combatant")
	}
	if len(slices.Compact(slices.Sorted(slices.Values(in.IDs)))) != len(in.IDs) {
		return InitiativeParticipant{}, invalidInput("a combatant appears twice in one Initiative entry")
	}
	if in.Surprise != NotSurprised && in.Surprise != Surprised {
		return InitiativeParticipant{}, invalidInput("surprise is not defined")
	}
	return InitiativeParticipant{slices.Clone(in.IDs), in.Modifier, in.RollOnly, in.Surprise,
		RollContributions{in.Contributions.Items()}, slices.Clone(in.Circumstances)}, nil
}

// IDs returns a copy of the combatants the entry names.
func (p InitiativeParticipant) IDs() []CombatantID { return slices.Clone(p.ids) }

// IsZero reports whether p is the invalid zero participant.
func (p InitiativeParticipant) IsZero() bool { return len(p.ids) == 0 }

// Mode resolves the roll mode over surprise, the circumstances inside the supplied
// RollContributions and the other circumstances together: repeated Advantage or
// Disadvantage does not stack and any of each cancels. It never uses
// RollContributions.Mode, which is already resolved (SRD 5.2.1, "They Don't
// Stack", p. 8).
func (p InitiativeParticipant) Mode() modifier.RollMode {
	var cs []modifier.RollCircumstance
	if p.surprise == Surprised {
		cs = append(cs, modifier.Disadvantage{})
	}
	for _, c := range p.contributions.Items() {
		cs = append(cs, c.Circumstance())
	}
	return modifier.Resolve(append(cs, p.others...)...)
}

// score returns the Initiative score: 10 plus the modifier, plus 5 with Advantage,
// minus 5 with Disadvantage; the roll-only modifier is ignored (SRD 5.2.1,
// "Initiative", p. 184).
func (p InitiativeParticipant) score() InitiativeCount {
	n := satAdd(10, p.mod.Value())
	switch p.Mode() {
	case modifier.Normal:
	case modifier.WithAdvantage:
		n = satAdd(n, 5)
	case modifier.WithDisadvantage:
		n = satAdd(n, -5)
	}
	return InitiativeCount(n)
}

// InitiativeBreakdown is how one participant's count was made, a closed set.
// Variants: RolledInitiative, ScoredInitiative.
type InitiativeBreakdown interface {
	isInitiativeBreakdown()
	// Method returns InitiativeMethodRolled or InitiativeMethodScore.
	Method() InitiativeMethod
	// IDs returns a copy of the combatants that received the count.
	IDs() []CombatantID
	// Mode returns the resolved roll mode.
	Mode() modifier.RollMode
	// Count returns the Initiative count.
	Count() InitiativeCount
}

// RolledInitiative: a rolled count, with the roll kept so a reroll or a replaced die can
// recompute it.
type RolledInitiative struct {
	ids                []CombatantID
	mode               modifier.RollMode
	roll               d20.Roll
	modifier, rollOnly modifier.Modifier
	count              InitiativeCount
}

// ScoredInitiative: an Initiative score; no die was rolled.
type ScoredInitiative struct {
	ids      []CombatantID
	mode     modifier.RollMode
	modifier modifier.Modifier
	count    InitiativeCount
}

func (RolledInitiative) isInitiativeBreakdown() {}
func (ScoredInitiative) isInitiativeBreakdown() {}

// Method returns InitiativeMethodRolled.
func (RolledInitiative) Method() InitiativeMethod { return InitiativeMethodRolled }

// Method returns InitiativeMethodScore.
func (ScoredInitiative) Method() InitiativeMethod { return InitiativeMethodScore }

// IDs returns a copy of the combatants that received the count.
func (b RolledInitiative) IDs() []CombatantID { return slices.Clone(b.ids) }

// IDs returns a copy of the combatants that received the count.
func (b ScoredInitiative) IDs() []CombatantID { return slices.Clone(b.ids) }

// Mode returns the resolved roll mode.
func (b RolledInitiative) Mode() modifier.RollMode { return b.mode }

// Mode returns the resolved roll mode.
func (b ScoredInitiative) Mode() modifier.RollMode { return b.mode }

// Count returns the Initiative count.
func (b RolledInitiative) Count() InitiativeCount { return b.count }

// Count returns the Initiative count.
func (b ScoredInitiative) Count() InitiativeCount { return b.count }

// Roll returns the d20 roll (every die in request order and the one used).
func (b RolledInitiative) Roll() d20.Roll { return b.roll }

// InitiativeModifier returns the Initiative modifier added.
func (b RolledInitiative) InitiativeModifier() modifier.Modifier { return b.modifier }

// RollOnlyModifier returns the roll-only modifier added.
func (b RolledInitiative) RollOnlyModifier() modifier.Modifier { return b.rollOnly }

// InitiativeModifier returns the Initiative modifier added to 10.
func (b ScoredInitiative) InitiativeModifier() modifier.Modifier { return b.modifier }
