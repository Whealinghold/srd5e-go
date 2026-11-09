package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// FighterEffects reports the base Fighter features whose effects other contexts
// apply: Tactical Shift, Tactical Master, Studied Attacks, the attack count and
// the Weapon Mastery access. They are queries over values; nothing is stored and
// no encounter state is held. A Fighter has them whatever the subclass. It is
// stateless; the zero value is ready to use. Safe for concurrent use.
type FighterEffects struct{}

// Fighter feature levels (SRD 5.2.1, "Fighter Features", p. 47).
const (
	tacticalShiftLevel  = 5
	tacticalMasterLevel = 9
	studiedAttacksLevel = 13
)

// FighterEffectTrigger is what sets a declared Fighter effect off. The zero value is not a trigger.
type FighterEffectTrigger int

// The effect triggers.
const (
	// TriggerAttackRollMissed: the character made an attack roll against a creature and missed.
	TriggerAttackRollMissed FighterEffectTrigger = iota + 1
	// TriggerStartOfEachOfYourTurns: the start of each of the character's turns.
	TriggerStartOfEachOfYourTurns
)

// String returns "attack_roll_missed", "start_of_each_of_your_turns" or "unknown".
func (t FighterEffectTrigger) String() string {
	switch t {
	case TriggerAttackRollMissed:
		return "attack_roll_missed"
	case TriggerStartOfEachOfYourTurns:
		return "start_of_each_of_your_turns"
	}
	return "unknown"
}

// FighterEffectTarget is what a declared Fighter effect applies to. The zero value is not a target.
type FighterEffectTarget int

// The effect targets.
const (
	// NextAttackRollAgainstSameCreature: the next attack roll against the creature that was missed.
	NextAttackRollAgainstSameCreature FighterEffectTarget = iota + 1
)

// String returns "next_attack_roll_against_same_creature" or "unknown".
func (t FighterEffectTarget) String() string {
	if t == NextAttackRollAgainstSameCreature {
		return "next_attack_roll_against_same_creature"
	}
	return "unknown"
}

// FighterEffectExpiry is when a declared Fighter effect ends. The zero value is not an expiry.
type FighterEffectExpiry int

// The effect expiries.
const (
	// ExpiresEndOfYourNextTurn: before the end of the character's next turn.
	ExpiresEndOfYourNextTurn FighterEffectExpiry = iota + 1
)

// String returns "end_of_your_next_turn" or "unknown".
func (e FighterEffectExpiry) String() string {
	if e == ExpiresEndOfYourNextTurn {
		return "end_of_your_next_turn"
	}
	return "unknown"
}

// StudiedAttacksEffect is the declared effect of Studied Attacks (SRD 5.2.1,
// "Level 13: Studied Attacks", p. 48): after a missed attack roll, Advantage on
// the next attack roll against that creature before the end of the character's
// next turn. It is data and holds no encounter state; the consumer applies it.
type StudiedAttacksEffect struct{}

// Trigger returns TriggerAttackRollMissed.
func (StudiedAttacksEffect) Trigger() FighterEffectTrigger { return TriggerAttackRollMissed }

// AppliesTo returns NextAttackRollAgainstSameCreature.
func (StudiedAttacksEffect) AppliesTo() FighterEffectTarget { return NextAttackRollAgainstSameCreature }

// Expires returns ExpiresEndOfYourNextTurn.
func (StudiedAttacksEffect) Expires() FighterEffectExpiry { return ExpiresEndOfYourNextTurn }

// StudiedAttacks reports Studied Attacks from Fighter level 13, for every Fighter
// whatever the subclass; false below it or for the zero level.
func (FighterEffects) StudiedAttacks(level FighterLevel) (StudiedAttacksEffect, bool) {
	return StudiedAttacksEffect{}, level.n >= studiedAttacksLevel
}

// TacticalShift reports whether Second Wind offers a move: TacticalShiftOffered
// from Fighter level 5 (SRD 5.2.1, "Level 5: Tactical Shift", p. 48), NoTacticalShift
// below it.
func (FighterEffects) TacticalShift(level FighterLevel) TacticalShiftOffer {
	if level.n >= tacticalShiftLevel {
		return TacticalShiftOffered{}
	}
	return NoTacticalShift{}
}

// AttacksPerAttackAction reports the attacks per Attack action as the attack count
// of stats, not as separate stacking effects (Extra Attack, Two Extra Attacks and
// Three Extra Attacks). It comes from the Fighter level alone.
func (FighterEffects) AttacksPerAttackAction(stats FighterLevelStats) int { return stats.attacks }

// MasteryAccess returns the WeaponMasteryAccess of a Fighter: NoMasteryFeature for
// a character with no Fighter levels or with no Weapon Mastery choice stored in
// states, and MasteryFromChoice for the stored choice otherwise. It returns an
// error matching srderr.ErrInvalidInput only when the weapon_mastery_choice name
// holds a state of another type. It never panics.
func (FighterEffects) MasteryAccess(classes ClassLevels, states FeatureStates) (WeaponMasteryAccess, error) {
	if classes.LevelIn(FighterClassID) < 1 {
		return NoMasteryFeature{}, nil
	}
	return WeaponMasteryAccessFor(states)
}

// TacticalMasterOffer is the closed result of FighterEffects.TacticalMasterOptions.
// Variants: TacticalMasterOffered, TacticalMasterUnavailable.
type TacticalMasterOffer interface{ isTacticalMasterOffer() }

// TacticalMasterOffered: the weapon's mastery property may be replaced, for that
// attack, by Push, Sap or Slow.
type TacticalMasterOffered struct{}

// Replacements returns Push, Sap and Slow, in that order, as a new slice. The
// stored Weapon Mastery choice is not changed by using one.
func (TacticalMasterOffered) Replacements() []content.MasteryProperty {
	return []content.MasteryProperty{content.MasteryPush, content.MasterySap, content.MasterySlow}
}

// TacticalMasterUnavailable: no replacement is offered. Construct it with a keyed literal.
type TacticalMasterUnavailable struct{ Reason FighterReason }

func (TacticalMasterOffered) isTacticalMasterOffer()     {}
func (TacticalMasterUnavailable) isTacticalMasterOffer() {}

// TacticalMasterOptions reports what Tactical Master offers for an attack with
// weapon (SRD 5.2.1, "Level 9: Tactical Master", p. 48): TacticalMasterOffered from
// Fighter level 9 when access permits the weapon, otherwise
// TacticalMasterUnavailable with FighterFeatureNotGained below level 9 (checked
// first) or FighterMasteryNotUsable when the kind is not chosen or access is nil or
// NoMasteryFeature.
func (FighterEffects) TacticalMasterOptions(level FighterLevel, access WeaponMasteryAccess, weapon content.ItemID) TacticalMasterOffer {
	switch {
	case level.n < tacticalMasterLevel:
		return TacticalMasterUnavailable{Reason: FighterFeatureNotGained}
	case access == nil || !access.Permits(weapon):
		return TacticalMasterUnavailable{Reason: FighterMasteryNotUsable}
	}
	return TacticalMasterOffered{}
}
