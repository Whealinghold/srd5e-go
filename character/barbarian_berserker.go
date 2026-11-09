package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// intimidatingPresenceBaseDC is the 8 of the Intimidating Presence Difficulty Class (SRD 5.2.1,
// "Level 14: Intimidating Presence", p. 30).
const intimidatingPresenceBaseDC = 8

// BerserkerFeaturesAt returns the ids of the Path of the Berserker features held at the Barbarian
// level when subclass is the Path of the Berserker: Frenzy from 3, Mindless Rage from 6,
// Retaliation from 10 and Intimidating Presence from 14 (SRD 5.2.1, "Barbarian Subclass: Path of
// the Berserker", p. 30), in level order, as a new slice. Any other subclass, or the zero level,
// holds none.
func BerserkerFeaturesAt(level BarbarianLevel, subclass content.SubclassID) []content.FeatureID {
	out := []content.FeatureID{}
	if subclass != BerserkerSubclassID {
		return out
	}
	levels := berserkerFeatureLevels()
	for id, l := range levels {
		if level.n >= l {
			out = append(out, id)
		}
	}
	slices.SortFunc(out, func(a, b content.FeatureID) int { return levels[a] - levels[b] })
	return out
}

// berserkerGate returns the reason the Path of the Berserker feature of minLevel is not available,
// or 0: BarbarianNotABarbarian, BarbarianFeatureNotGained, BarbarianNotABerserker, in that order.
func berserkerGate(stats BarbarianLevelStats, subclass content.SubclassID, minLevel int) BarbarianReason {
	switch {
	case stats.IsZero():
		return BarbarianNotABarbarian
	case stats.level.n < minLevel:
		return BarbarianFeatureNotGained
	case subclass != BerserkerSubclassID:
		return BarbarianNotABerserker
	}
	return 0
}

// FrenzyOutcome is the closed outcome of BerserkerRules.ApplyFrenzy. Variants: FrenzyApplied,
// FrenzyNotApplicable.
type FrenzyOutcome interface{ isFrenzyOutcome() }

// FrenzyApplied is Frenzy applying to the first target hit this turn. Only BerserkerRules.ApplyFrenzy
// builds one.
type FrenzyApplied struct {
	extra BarbarianExtraDamage
	turn  BarbarianTurnState
}

// ExtraDamage returns the extra damage to roll: d6s equal to the Rage Damage bonus, of the damage type
// of the attack. The dice are returned, not rolled.
func (r FrenzyApplied) ExtraDamage() BarbarianExtraDamage { return r.extra }

// Turn returns the new turn state, with the Frenzy target recorded.
func (r FrenzyApplied) Turn() BarbarianTurnState { return r.turn }

// FrenzyNotApplicable is a Frenzy that does not apply. Construct it with a keyed literal.
type FrenzyNotApplicable struct{ Reason BarbarianReason }

func (FrenzyApplied) isFrenzyOutcome()       {}
func (FrenzyNotApplicable) isFrenzyOutcome() {}

// RetaliationOutcome is the closed outcome of BerserkerRules.OfferRetaliation. Variants:
// RetaliationOffered, RetaliationNotOffered.
type RetaliationOutcome interface{ isRetaliationOutcome() }

// RetaliationOffered: the Barbarian may use its Reaction to make one melee attack against the creature
// that damaged it, with a weapon or an Unarmed Strike. Offering spends nothing.
type RetaliationOffered struct{}

// RetaliationNotOffered is a Retaliation that is not offered. Construct it with a keyed literal.
type RetaliationNotOffered struct{ Reason BarbarianReason }

func (RetaliationOffered) isRetaliationOutcome()    {}
func (RetaliationNotOffered) isRetaliationOutcome() {}

// IntimidatingPresenceOutcome is the closed outcome of BerserkerRules.UseIntimidatingPresence. Variants:
// IntimidatingPresenceApplied, IntimidatingPresenceRejected.
type IntimidatingPresenceOutcome interface{ isIntimidatingPresenceOutcome() }

// IntimidatingPresenceRestoration is the closed outcome of BerserkerRules.RestoreIntimidatingPresence.
// Variants: IntimidatingPresenceRestored, IntimidatingPresenceRejected.
type IntimidatingPresenceRestoration interface{ isIntimidatingPresenceRestoration() }

// IntimidatingPresenceApplied is a use of Intimidating Presence. Only BerserkerRules builds one.
type IntimidatingPresenceApplied struct {
	dc     int
	state  BarbarianState
	turn   BarbarianTurnState
	events []Event
}

// DC returns the Wisdom saving throw Difficulty Class: 8 plus the Strength modifier plus the Proficiency Bonus.
func (r IntimidatingPresenceApplied) DC() int { return r.dc }

// State returns the new state: Intimidating Presence spent until a Long Rest or a restoration.
func (r IntimidatingPresenceApplied) State() BarbarianState { return r.state }

// Turn returns the new turn state: the Bonus Action spent.
func (r IntimidatingPresenceApplied) Turn() BarbarianTurnState { return r.turn }

// Events returns a new slice holding the IntimidatingPresenceUsed event.
func (r IntimidatingPresenceApplied) Events() []Event { return slices.Clone(r.events) }

// IntimidatingPresenceRestored is a restoration of Intimidating Presence by expending a Rage use.
type IntimidatingPresenceRestored struct{ state BarbarianState }

// State returns the new state: a Rage use expended and Intimidating Presence available again.
func (r IntimidatingPresenceRestored) State() BarbarianState { return r.state }

// IntimidatingPresenceRejected is a use or restoration a rule refused: nothing is spent. Construct it
// with a keyed literal.
type IntimidatingPresenceRejected struct{ Reason BarbarianReason }

func (IntimidatingPresenceApplied) isIntimidatingPresenceOutcome()      {}
func (IntimidatingPresenceRejected) isIntimidatingPresenceOutcome()     {}
func (IntimidatingPresenceRestored) isIntimidatingPresenceRestoration() {}
func (IntimidatingPresenceRejected) isIntimidatingPresenceRestoration() {}

// BerserkerRules decides the Path of the Berserker features (SRD 5.2.1, "Barbarian Subclass: Path of
// the Berserker", p. 30). It rolls no dice. It is stateless and pure; the zero value is ready to use.
// Safe for concurrent use.
type BerserkerRules struct{}

// ApplyFrenzy decides Frenzy for attack. The first reason that holds refuses it, in this order:
// BarbarianNotABarbarian, BarbarianFeatureNotGained (below level 3), BarbarianNotABerserker,
// BarbarianRageNotActive, BarbarianRecklessNotDeclared, BarbarianNotAStrengthAttack,
// BarbarianAttackMissed, BarbarianFrenzyTargetAlreadyHit. Otherwise it returns d6s equal to the
// Rage Damage bonus, not rolled, and records the target.
func (BerserkerRules) ApplyFrenzy(stats BarbarianLevelStats, subclass content.SubclassID, state BarbarianState, turn BarbarianTurnState, attack BarbarianAttackFacts) FrenzyOutcome {
	reason := berserkerGate(stats, subclass, frenzySubclassLevel)
	switch {
	case reason != 0:
	case !state.rageActive:
		reason = BarbarianRageNotActive
	case !turn.recklessDeclared:
		reason = BarbarianRecklessNotDeclared
	case !attack.UsesStrength:
		reason = BarbarianNotAStrengthAttack
	case !attack.Hit:
		reason = BarbarianAttackMissed
	case turn.frenzyTargetHit:
		reason = BarbarianFrenzyTargetAlreadyHit
	}
	if reason != 0 {
		return FrenzyNotApplicable{Reason: reason}
	}
	return FrenzyApplied{
		extra: BarbarianExtraDamage{count: stats.rageDamage, die: frenzySides, damageType: attack.DamageType}, turn: turn.RecordFrenzyTarget(),
	}
}

// OfferRetaliation decides Retaliation when the Barbarian takes damage. The first reason that holds
// refuses it, in this order: BarbarianNotABarbarian, BarbarianFeatureNotGained (below level 10),
// BarbarianNotABerserker, BarbarianSourceNotWithinFiveFeet, BarbarianNoReaction.
func (BerserkerRules) OfferRetaliation(stats BarbarianLevelStats, subclass content.SubclassID, turn BarbarianTurnState, sourceWithinFiveFeet bool) RetaliationOutcome {
	reason := berserkerGate(stats, subclass, retaliationLevel)
	switch {
	case reason != 0:
	case !sourceWithinFiveFeet:
		reason = BarbarianSourceNotWithinFiveFeet
	case !turn.reactionAvailable:
		reason = BarbarianNoReaction
	}
	if reason != 0 {
		return RetaliationNotOffered{Reason: reason}
	}
	return RetaliationOffered{}
}

// UseIntimidatingPresence uses Intimidating Presence as a Bonus Action. The first reason that holds
// rejects it, in this order: BarbarianNotABarbarian, BarbarianFeatureNotGained (below level 14),
// BarbarianNotABerserker, BarbarianIncapacitated, BarbarianNoBonusAction, BarbarianPresenceNotRestored.
// The DC is 8 plus strengthModifier plus proficiency, the Proficiency Bonus of the total character
// level, not of the Barbarian level (SRD 5.2.1, "Multiclassing", p. 25). It returns an error matching
// srderr.ErrInvalidInput for a zero id or a zero proficiency bonus.
func (BerserkerRules) UseIntimidatingPresence(id Identifier, stats BarbarianLevelStats, subclass content.SubclassID, state BarbarianState, turn BarbarianTurnState,
	facts BarbarianHostFacts, strengthModifier ability.Modifier, proficiency ProficiencyBonus) (IntimidatingPresenceOutcome, error) {
	if id.IsZero() || proficiency.IsZero() {
		return nil, invalidInput("use Intimidating Presence with a zero character id or Proficiency Bonus")
	}
	reason := berserkerGate(stats, subclass, intimidatingPresenceLvl)
	switch {
	case reason != 0:
	case facts.Incapacitated:
		reason = BarbarianIncapacitated
	case !turn.bonusActionAvailable:
		reason = BarbarianNoBonusAction
	case state.presenceUsed:
		reason = BarbarianPresenceNotRestored
	}
	if reason != 0 {
		return IntimidatingPresenceRejected{Reason: reason}, nil
	}
	dc := intimidatingPresenceBaseDC + strengthModifier.Value() + proficiency.Value()
	state.presenceUsed = true
	return IntimidatingPresenceApplied{dc: dc, state: state, turn: turn.SpendBonusAction(), events: []Event{IntimidatingPresenceUsed{id: id, dc: dc}}}, nil
}

// RestoreIntimidatingPresence restores a spent Intimidating Presence by expending a Rage use, with no
// action. The first reason that holds rejects it, in this order: BarbarianNotABarbarian,
// BarbarianFeatureNotGained, BarbarianNotABerserker, BarbarianPresenceNotUsed,
// BarbarianNoRageUsesRemaining.
func (BerserkerRules) RestoreIntimidatingPresence(stats BarbarianLevelStats, subclass content.SubclassID, state BarbarianState) IntimidatingPresenceRestoration {
	reason := berserkerGate(stats, subclass, intimidatingPresenceLvl)
	switch {
	case reason != 0:
	case !state.presenceUsed:
		reason = BarbarianPresenceNotUsed
	case state.RagesRemaining(stats) < 1:
		reason = BarbarianNoRageUsesRemaining
	}
	if reason != 0 {
		return IntimidatingPresenceRejected{Reason: reason}
	}
	state.ragesExpended++
	state.presenceUsed = false
	return IntimidatingPresenceRestored{state: state}
}
