package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// The Rogue levels of the features in this file (SRD 5.2.1, "Rogue Features", pp. 62-63).
const (
	cunningActionLevel  = 2
	steadyAimLevel      = 3
	uncannyDodgeLevel   = 5
	evasionLevel        = 7
	reliableTalentLevel = 7
	elusiveLevel        = 18
	reliableTalentFloor = 10
)

// RogueGrantedAction is an action Cunning Action lets the Rogue take as a Bonus Action (SRD 5.2.1, "Level 2: Cunning
// Action", p. 62). The zero value is invalid.
type RogueGrantedAction int

// The actions Cunning Action grants.
const (
	RogueDash RogueGrantedAction = iota + 1
	RogueDisengage
	RogueHide
)

// String returns "dash", "disengage", "hide" or "unknown".
func (a RogueGrantedAction) String() string {
	switch a {
	case RogueDash:
		return "dash"
	case RogueDisengage:
		return "disengage"
	case RogueHide:
		return "hide"
	}
	return "unknown"
}

// RogueGrantedActions is the Cunning Action rule. The zero value is ready to use.
type RogueGrantedActions struct{}

// At returns the Bonus Action options Cunning Action gives at the Rogue level: Dash, Disengage and Hide from level 2, none at
// level 1 (SRD 5.2.1, "Level 2: Cunning Action", p. 62).
func (RogueGrantedActions) At(level RogueLevel) []RogueGrantedAction {
	if level.n < cunningActionLevel {
		return []RogueGrantedAction{}
	}
	return []RogueGrantedAction{RogueDash, RogueDisengage, RogueHide}
}

// SteadyAimReason is why Steady Aim is not available. The values are in check order. The zero value is not a reason.
type SteadyAimReason int

const (
	// SteadyAimLevelTooLow: the Rogue level is below 3.
	SteadyAimLevelTooLow SteadyAimReason = iota + 1
	// SteadyAimMovedThisTurn: the character has moved this turn.
	SteadyAimMovedThisTurn
)

// SteadyAimResult is the outcome of SteadyAim.Use: the closed set SteadyAimGranted and SteadyAimNotAvailable.
//
//sumtype:decl
type SteadyAimResult interface{ isSteadyAimResult() }

// SteadyAimAdvantageOn says which roll Steady Aim gives Advantage on; the only value is the next attack roll this turn.
type SteadyAimAdvantageOn int

// SteadyAimNextAttackRollThisTurn is the roll Steady Aim gives Advantage on.
const SteadyAimNextAttackRollThisTurn SteadyAimAdvantageOn = iota + 1

// SteadyAimActionCost is what Steady Aim costs to use; the only value is a Bonus Action.
type SteadyAimActionCost int

// SteadyAimBonusAction is the action cost of Steady Aim.
const SteadyAimBonusAction SteadyAimActionCost = iota + 1

// SteadyAimGranted means Steady Aim may be used. The caller applies the effects.
type SteadyAimGranted struct{}

// AdvantageOn returns the roll that gets Advantage.
func (SteadyAimGranted) AdvantageOn() SteadyAimAdvantageOn { return SteadyAimNextAttackRollThisTurn }

// SpeedUntilEndOfTurn returns the Speed in feet until the end of the current turn, 0.
func (SteadyAimGranted) SpeedUntilEndOfTurn() int { return 0 }

// ActionCost returns what the feature costs, a Bonus Action.
func (SteadyAimGranted) ActionCost() SteadyAimActionCost { return SteadyAimBonusAction }

// SteadyAimNotAvailable means Steady Aim may not be used, with every failing reason in check order.
type SteadyAimNotAvailable struct{ reasons []SteadyAimReason }

// Reasons returns the failing reasons as a new slice, in check order.
func (r SteadyAimNotAvailable) Reasons() []SteadyAimReason { return slices.Clone(r.reasons) }

func (SteadyAimGranted) isSteadyAimResult()      {}
func (SteadyAimNotAvailable) isSteadyAimResult() {}

// SteadyAim is the Steady Aim rule (SRD 5.2.1, "Level 3: Steady Aim", p. 62). The zero value is ready to use.
type SteadyAim struct{}

// Use decides whether the Rogue can use Steady Aim: from Rogue level 3, and only if the character has not moved this turn.
func (SteadyAim) Use(level RogueLevel, movedThisTurn bool) SteadyAimResult {
	var reasons []SteadyAimReason
	if level.n < steadyAimLevel {
		reasons = append(reasons, SteadyAimLevelTooLow)
	}
	if movedThisTurn {
		reasons = append(reasons, SteadyAimMovedThisTurn)
	}
	if len(reasons) > 0 {
		return SteadyAimNotAvailable{reasons: reasons}
	}
	return SteadyAimGranted{}
}

// RogueUncannyDodgeReason is why Uncanny Dodge is not available. The values are in check order. The zero value is not a reason.
type RogueUncannyDodgeReason int

const (
	// RogueUncannyDodgeLevelTooLow: the Rogue level is below 5.
	RogueUncannyDodgeLevelTooLow RogueUncannyDodgeReason = iota + 1
	// RogueUncannyDodgeReactionUnavailable: the Reaction is not available.
	RogueUncannyDodgeReactionUnavailable
	// RogueUncannyDodgeAttackerNotSeen: the attacker is not a creature the character can see.
	RogueUncannyDodgeAttackerNotSeen
	// RogueUncannyDodgeAttackMissed: the attack did not hit.
	RogueUncannyDodgeAttackMissed
)

// RogueUncannyDodgeResult is the outcome of RogueUncannyDodge.CanUse: the closed set RogueUncannyDodgeAvailable and
// RogueUncannyDodgeNotAvailable.
//
//sumtype:decl
type RogueUncannyDodgeResult interface{ isRogueUncannyDodgeResult() }

// RogueUncannyDodgeAvailable means the Rogue can take the Reaction.
type RogueUncannyDodgeAvailable struct{}

// RogueUncannyDodgeNotAvailable means the Rogue cannot, with every failing reason in check order.
type RogueUncannyDodgeNotAvailable struct{ reasons []RogueUncannyDodgeReason }

// Reasons returns the failing reasons as a new slice, in check order.
func (r RogueUncannyDodgeNotAvailable) Reasons() []RogueUncannyDodgeReason {
	return slices.Clone(r.reasons)
}

func (RogueUncannyDodgeAvailable) isRogueUncannyDodgeResult()    {}
func (RogueUncannyDodgeNotAvailable) isRogueUncannyDodgeResult() {}

// RogueUncannyDodge is the Uncanny Dodge rule (SRD 5.2.1, "Level 5: Uncanny Dodge", p. 63). The zero value is ready to use.
type RogueUncannyDodge struct{}

// Halve returns the damage halved and rounded down. It returns an error matching srderr.ErrInvalidInput for negative damage.
func (RogueUncannyDodge) Halve(damage int) (int, error) {
	if damage < 0 {
		return 0, invalidInput("negative damage")
	}
	return damage / 2, nil
}

// CanUse decides whether the Rogue can use Uncanny Dodge: from Rogue level 5, with the Reaction available, against an
// attacker the character can see whose attack roll hit.
func (RogueUncannyDodge) CanUse(level RogueLevel, reactionAvailable, attackerSeen bool, attack d20.AttackRollResult) RogueUncannyDodgeResult {
	var reasons []RogueUncannyDodgeReason
	if level.n < uncannyDodgeLevel {
		reasons = append(reasons, RogueUncannyDodgeLevelTooLow)
	}
	if !reactionAvailable {
		reasons = append(reasons, RogueUncannyDodgeReactionUnavailable)
	}
	if !attackerSeen {
		reasons = append(reasons, RogueUncannyDodgeAttackerNotSeen)
	}
	if !attack.Hit() {
		reasons = append(reasons, RogueUncannyDodgeAttackMissed)
	}
	if len(reasons) > 0 {
		return RogueUncannyDodgeNotAvailable{reasons: reasons}
	}
	return RogueUncannyDodgeAvailable{}
}

// RogueEvasionInput is what RogueEvasion.Apply reads. Construct it with a keyed literal.
type RogueEvasionInput struct {
	// Level is the Rogue level.
	Level RogueLevel
	// FullDamage is the effect's damage on a failed saving throw.
	FullDamage int
	// SaveAbility is the ability of the saving throw.
	SaveAbility ability.Ability
	// HalfOnSuccess reports that a success takes only half damage; false when a success takes none.
	HalfOnSuccess bool
	// SaveSucceeded reports that the saving throw succeeded.
	SaveSucceeded bool
	// Incapacitated reports that the character has the Incapacitated condition.
	Incapacitated bool
}

// RogueEvasion is the Evasion rule (SRD 5.2.1, "Level 7: Evasion", p. 63). The zero value is ready to use.
type RogueEvasion struct{}

// Apply returns the damage the Rogue takes. For a Dexterity saving throw against half damage on a success, from Rogue level 7
// and while not Incapacitated, a success takes none and a failure half, rounded down; otherwise the normal result stands: the
// full damage on a failure, and on a success half or none as the effect says. It returns an error matching
// srderr.ErrInvalidInput for negative damage.
func (RogueEvasion) Apply(in RogueEvasionInput) (int, error) {
	if in.FullDamage < 0 {
		return 0, invalidInput("negative damage")
	}
	normal := in.FullDamage
	if in.SaveSucceeded {
		normal = 0
		if in.HalfOnSuccess {
			normal = in.FullDamage / 2
		}
	}
	if in.SaveAbility != ability.Dexterity || !in.HalfOnSuccess || in.Level.n < evasionLevel || in.Incapacitated {
		return normal, nil
	}
	if in.SaveSucceeded {
		return 0, nil
	}
	return in.FullDamage / 2, nil
}

// RogueReliableTalent is the Reliable Talent rule (SRD 5.2.1, "Level 7: Reliable Talent", p. 63). The zero value is ready to use.
type RogueReliableTalent struct{}

// Apply returns the d20 roll to use: 10 for a roll of 9 or lower, the roll otherwise, for an ability check that uses one of the
// character's skill or tool proficiencies at Rogue level 7 or higher. Any other roll is unchanged. It returns an error matching
// srderr.ErrInvalidInput for a roll outside 1 to 20.
func (RogueReliableTalent) Apply(level RogueLevel, kind d20.Kind, usesProficiency bool, roll int) (int, error) {
	if roll < 1 || roll > 20 {
		return 0, invalidInput("d20 roll outside 1 to 20")
	}
	if level.n >= reliableTalentLevel && kind == d20.AbilityCheck && usesProficiency && roll < reliableTalentFloor {
		return reliableTalentFloor, nil
	}
	return roll, nil
}

// RogueElusive is the Elusive rule (SRD 5.2.1, "Level 18: Elusive", p. 63). The zero value is ready to use.
type RogueElusive struct{}

// Apply returns the circumstances of an attack roll against the character after Elusive: from Rogue level 18 every Advantage is
// dropped and every Disadvantage kept, unless the character is Incapacitated. Otherwise the circumstances are returned
// unchanged. The input slice is not changed; the result is a new slice, to resolve with modifier.Resolve.
func (RogueElusive) Apply(level RogueLevel, incapacitated bool, circumstances []modifier.RollCircumstance) []modifier.RollCircumstance {
	if level.n < elusiveLevel || incapacitated {
		return slices.Clone(circumstances)
	}
	out := make([]modifier.RollCircumstance, 0, len(circumstances))
	for _, c := range circumstances {
		if _, adv := c.(modifier.Advantage); !adv {
			out = append(out, c)
		}
	}
	return out
}
