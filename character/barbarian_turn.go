package character

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// BarbarianTurnState is what a Barbarian has done this turn: whether Reckless
// Attack is declared, whether the first attack roll is made, whether a target was
// hit for Frenzy, and whether the Bonus Action and the Reaction are still available.
// It is per-turn host state, not stored on the character; the commands of the
// functions that need it carry it. The only home of Bonus Action and Reaction
// availability. Immutable, comparable with ==; the zero value is a spent turn (no
// Bonus Action, no Reaction): build a fresh turn with NewBarbarianTurn. Safe for
// concurrent use.
type BarbarianTurnState struct {
	recklessDeclared, firstAttackRollMade, frenzyTargetHit bool
	bonusActionAvailable, reactionAvailable                bool
}

// NewBarbarianTurn returns the state at the start of the Barbarian's turn: nothing
// declared, no attack rolled, the Bonus Action and the Reaction available.
func NewBarbarianTurn() BarbarianTurnState {
	return BarbarianTurnState{bonusActionAvailable: true, reactionAvailable: true}
}

// StartNextTurn returns the state at the start of the Barbarian's next turn. Reckless
// Attack lasts until that moment (SRD 5.2.1, "Level 2: Reckless Attack", p. 29), so it
// is cleared along with every other per-turn flag.
func (BarbarianTurnState) StartNextTurn() BarbarianTurnState { return NewBarbarianTurn() }

// RecordAttackRoll returns the state after the first or a later attack roll: the first
// attack roll of the turn is made.
func (t BarbarianTurnState) RecordAttackRoll() BarbarianTurnState {
	t.firstAttackRollMade = true
	return t
}

// RecordFrenzyTarget returns the state after the first target was hit for Frenzy.
func (t BarbarianTurnState) RecordFrenzyTarget() BarbarianTurnState {
	t.frenzyTargetHit = true
	return t
}

// DeclareReckless returns the state with Reckless Attack declared.
func (t BarbarianTurnState) DeclareReckless() BarbarianTurnState {
	t.recklessDeclared = true
	return t
}

// SpendBonusAction returns the state with the Bonus Action spent.
func (t BarbarianTurnState) SpendBonusAction() BarbarianTurnState {
	t.bonusActionAvailable = false
	return t
}

// SpendReaction returns the state with the Reaction spent.
func (t BarbarianTurnState) SpendReaction() BarbarianTurnState {
	t.reactionAvailable = false
	return t
}

// RecklessDeclared reports whether Reckless Attack is declared since the last start of the turn.
func (t BarbarianTurnState) RecklessDeclared() bool { return t.recklessDeclared }

// FirstAttackRollMade reports whether the first attack roll of the turn is made.
func (t BarbarianTurnState) FirstAttackRollMade() bool { return t.firstAttackRollMade }

// FrenzyTargetHit reports whether a target was already hit for Frenzy this turn.
func (t BarbarianTurnState) FrenzyTargetHit() bool { return t.frenzyTargetHit }

// BonusActionAvailable reports whether the Bonus Action is available.
func (t BarbarianTurnState) BonusActionAvailable() bool { return t.bonusActionAvailable }

// ReactionAvailable reports whether the Reaction is available.
func (t BarbarianTurnState) ReactionAvailable() bool { return t.reactionAvailable }

// BarbarianHostFacts are the facts about the character the host knows and the
// character context cannot: they are character-owned values, never combat types.
// The Bonus Action and the Reaction are not here but in BarbarianTurnState, so the two
// cannot disagree. Construct it with a keyed literal; the zero value says none holds.
type BarbarianHostFacts struct {
	Incapacitated, Unconscious, WearingHeavyArmor bool
	// DroppedToZeroHitPoints says the character dropped to 0 Hit Points and did not die outright.
	DroppedToZeroHitPoints bool
	SourceWithinFiveFeet   bool
	// ActiveConditions are the conditions the character has; the function reads a copy.
	ActiveConditions []content.ConditionKind
}

// BarbarianAttackKind is the kind of attack: a weapon attack or an Unarmed Strike.
// The zero value is not a kind.
type BarbarianAttackKind int

// The attack kinds.
const (
	// BarbarianWeaponAttack: an attack with a weapon.
	BarbarianWeaponAttack BarbarianAttackKind = iota + 1
	// BarbarianUnarmedStrike: an Unarmed Strike.
	BarbarianUnarmedStrike
)

// String returns "weapon", "unarmed_strike" or "unknown".
func (k BarbarianAttackKind) String() string {
	switch k {
	case BarbarianWeaponAttack:
		return "weapon"
	case BarbarianUnarmedStrike:
		return "unarmed_strike"
	}
	return "unknown"
}

// BarbarianAttackFacts describe one attack roll for the Barbarian rules: whether it
// uses Strength, its kind, whether it deals damage, the damage type of the weapon or
// Unarmed Strike, the RollMode after every Advantage and Disadvantage was combined (an
// attack roll with both has neither and is Normal) and whether it hit. Construct it with
// a keyed literal.
type BarbarianAttackFacts struct {
	UsesStrength bool
	Kind         BarbarianAttackKind
	DealsDamage  bool
	DamageType   content.DamageType
	RollMode     modifier.RollMode
	Hit          bool
}

// BarbarianExtraDamage is extra damage the host rolls: the dice and the damage type.
// On a Critical Hit the caller that rolls the damage doubles the dice (SRD 5.2.1,
// "Critical Hit", p. 179). Only the Barbarian functions build one.
type BarbarianExtraDamage struct {
	count, die int
	damageType content.DamageType
}

// Count returns how many dice.
func (d BarbarianExtraDamage) Count() int { return d.count }

// Sides returns how many sides each die has.
func (d BarbarianExtraDamage) Sides() int { return d.die }

// DamageType returns the damage type of the weapon or Unarmed Strike.
func (d BarbarianExtraDamage) DamageType() content.DamageType { return d.damageType }

// RageEndCause is why a Rage may end, a closed set. Variants: RageTurnEnded,
// RageHeavyArmorDonned, RageBecameIncapacitated, RageBecameUnconscious,
// RageElapsedTime.
type RageEndCause interface{ isRageEndCause() }

// RageTurnEnded: the Barbarian's turn ended.
type RageTurnEnded struct{}

// RageHeavyArmorDonned: the Barbarian donned Heavy armor.
type RageHeavyArmorDonned struct{}

// RageBecameIncapacitated: the Barbarian gained the Incapacitated condition without Unconscious.
type RageBecameIncapacitated struct{}

// RageBecameUnconscious: the Barbarian gained the Unconscious condition.
type RageBecameUnconscious struct{}

// RageElapsedTime: the Rage has been maintained for Elapsed, supplied by the host; the
// character context reads no clock. Construct it with a keyed literal.
type RageElapsedTime struct{ Elapsed time.Duration }

func (RageTurnEnded) isRageEndCause()           {}
func (RageHeavyArmorDonned) isRageEndCause()    {}
func (RageBecameIncapacitated) isRageEndCause() {}
func (RageBecameUnconscious) isRageEndCause()   {}
func (RageElapsedTime) isRageEndCause()         {}

// RageEndReason is the reason recorded on a RageEnded event. The zero value is not a
// reason.
type RageEndReason int

// The reasons a Rage ends.
const (
	// RageNotExtended: the Rage was not extended on the Barbarian's turn.
	RageNotExtended RageEndReason = iota + 1
	// RageEndedByHeavyArmor: the Barbarian donned Heavy armor.
	RageEndedByHeavyArmor
	// RageEndedByIncapacitated: the Barbarian became Incapacitated (or Unconscious below level 15).
	RageEndedByIncapacitated
	// RageEndedByUnconscious: the Barbarian became Unconscious (from level 15).
	RageEndedByUnconscious
	// RageEndedByTenMinutes: 10 minutes have elapsed.
	RageEndedByTenMinutes
)

// String returns "not_extended", "heavy_armor_donned", "incapacitated", "unconscious",
// "ten_minutes_elapsed" or "unknown".
func (r RageEndReason) String() string {
	names := []string{"not_extended", "heavy_armor_donned", "incapacitated", "unconscious", "ten_minutes_elapsed"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// hasCondition reports whether conditions holds kind.
func hasCondition(conditions []content.ConditionKind, kind content.ConditionKind) bool {
	return slices.Contains(conditions, kind)
}
