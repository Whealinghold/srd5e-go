package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// The Rogue levels that unlock Cunning Strike options (SRD 5.2.1, pp. 63-64).
const (
	cunningStrikeLevel         = 5
	improvedCunningStrikeLevel = 11
	deviousStrikesLevel        = 14
	supremeSneakLevel          = 9
	cunningStrikeDCBase        = 8
	largestTripTarget          = content.SizeLarge
)

// CunningStrikeEffect is one Cunning Strike effect (SRD 5.2.1, "Level 5: Cunning Strike", "Level 14: Devious Strikes", "Level
// 9: Supreme Sneak", pp. 63-64). The zero value is invalid.
type CunningStrikeEffect int

// The Cunning Strike effects.
const (
	CunningStrikePoison CunningStrikeEffect = iota + 1
	CunningStrikeTrip
	CunningStrikeWithdraw
	CunningStrikeDaze
	CunningStrikeKnockOut
	CunningStrikeObscure
	CunningStrikeStealthAttack
)

// String returns the effect as snake_case, or "unknown".
func (e CunningStrikeEffect) String() string {
	names := [...]string{"poison", "trip", "withdraw", "daze", "knock_out", "obscure", "stealth_attack"}
	if e < CunningStrikePoison || e > CunningStrikeStealthAttack {
		return "unknown"
	}
	return names[e-1]
}

// Cost returns the die cost of the effect in d6; 0 for an invalid effect.
func (e CunningStrikeEffect) Cost() SneakAttackDice {
	costs := [...]int{1, 1, 1, 2, 6, 3, 1}
	if e < CunningStrikePoison || e > CunningStrikeStealthAttack {
		return SneakAttackDice{}
	}
	return SneakAttackDice{n: costs[e-1]}
}

// availableTo reports whether a Rogue of level, with subclass, has the effect.
func (e CunningStrikeEffect) availableTo(level RogueLevel, subclass content.SubclassID) bool {
	switch e {
	case CunningStrikePoison, CunningStrikeTrip, CunningStrikeWithdraw:
		return level.n >= cunningStrikeLevel
	case CunningStrikeDaze, CunningStrikeKnockOut, CunningStrikeObscure:
		return level.n >= deviousStrikesLevel
	case CunningStrikeStealthAttack:
		return subclass == ThiefSubclassID && level.n >= supremeSneakLevel
	}
	return false
}

// CunningStrikeProblem is why a Cunning Strike selection was rejected. The values are in check order. The zero value is not a problem.
type CunningStrikeProblem int

const (
	// CunningStrikeSneakAttackNotDealt: a non-empty selection without Sneak Attack damage.
	CunningStrikeSneakAttackNotDealt CunningStrikeProblem = iota + 1
	// CunningStrikeTooManyEffects: more than one effect below Rogue level 11, more than two from it.
	CunningStrikeTooManyEffects
	// CunningStrikeDuplicateEffect: the same effect twice.
	CunningStrikeDuplicateEffect
	// CunningStrikeEffectNotAvailable: the Rogue level or subclass does not give the effect, or it is not an effect.
	CunningStrikeEffectNotAvailable
	// CunningStrikeInsufficientDice: the total cost exceeds the Sneak Attack dice.
	CunningStrikeInsufficientDice
	// CunningStrikePoisonersKitMissing: Poison without a Poisoner's Kit on the character's person.
	CunningStrikePoisonersKitMissing
)

// String returns the problem as snake_case, or "unknown".
func (p CunningStrikeProblem) String() string {
	names := [...]string{"sneak_attack_not_dealt", "too_many_effects", "duplicate_effect", "effect_not_available", "insufficient_dice", "poisoners_kit_missing"}
	if p < CunningStrikeSneakAttackNotDealt || p > CunningStrikePoisonersKitMissing {
		return "unknown"
	}
	return names[p-1]
}

// CunningStrikeTiming says when an effect occurs; the only value is immediately after the attack's damage is dealt.
type CunningStrikeTiming int

// CunningStrikeImmediatelyAfterDamage is the timing of every effect (SRD 5.2.1, "Level 5: Cunning Strike", p. 63).
const CunningStrikeImmediatelyAfterDamage CunningStrikeTiming = iota + 1

// CunningStrikeDuration is how long a condition from a Cunning Strike lasts.
type CunningStrikeDuration int

// The durations.
const (
	// CunningStrikeOneMinute: 1 minute.
	CunningStrikeOneMinute CunningStrikeDuration = iota + 1
	// CunningStrikeUntilEndOfTargetsNextTurn: until the end of the target's next turn.
	CunningStrikeUntilEndOfTargetsNextTurn
	// CunningStrikeNoneSpecified: the SRD states no duration.
	CunningStrikeNoneSpecified
)

// CunningStrikeRepeatTrigger is when a target repeats its saving throw; the only value is the end of each of its turns.
type CunningStrikeRepeatTrigger int

// CunningStrikeEndOfEachOfTargetsTurns is the repeat trigger of Poison and Knock Out.
const CunningStrikeEndOfEachOfTargetsTurns CunningStrikeRepeatTrigger = iota + 1

// CunningStrikeRepeatSaveRule is whether the target repeats the saving throw: the closed set CunningStrikeNoRepeatSave and
// CunningStrikeRepeatSave.
//
//sumtype:decl
type CunningStrikeRepeatSaveRule interface{ isCunningStrikeRepeatSaveRule() }

// CunningStrikeNoRepeatSave means the target does not repeat the saving throw.
type CunningStrikeNoRepeatSave struct{}

// CunningStrikeRepeatSave means the target repeats the saving throw of Ability at Trigger; EndsOnSuccess says that a success
// ends the effect on itself.
type CunningStrikeRepeatSave struct {
	Ability       ability.Ability
	Trigger       CunningStrikeRepeatTrigger
	EndsOnSuccess bool
}

func (CunningStrikeNoRepeatSave) isCunningStrikeRepeatSaveRule() {}
func (CunningStrikeRepeatSave) isCunningStrikeRepeatSaveRule()   {}

// CunningStrikeCommon is what every outcome carries: the effect, its die cost and when it occurs.
type CunningStrikeCommon struct {
	Effect CunningStrikeEffect
	Cost   SneakAttackDice
	Timing CunningStrikeTiming
}

// CunningStrikeOutcome is one selected effect, as a value for the caller to apply: the closed set
// CunningStrikeConditionOnFailedSave, CunningStrikeDazeOnFailedSave, CunningStrikeWithdrawMovement,
// CunningStrikeStealthAttackKeepsInvisible and CunningStrikeNoEffect. It carries no display text.
//
//sumtype:decl
type CunningStrikeOutcome interface{ Common() CunningStrikeCommon }

// CunningStrikeConditionOnFailedSave: on a failed saving throw the target gets Condition for Duration.
type CunningStrikeConditionOnFailedSave struct {
	CunningStrikeCommon
	SaveAbility  ability.Ability
	DC           d20.DifficultyClass
	Condition    content.ConditionKind
	Duration     CunningStrikeDuration
	RepeatSave   CunningStrikeRepeatSaveRule
	EndsOnDamage bool
}

// CunningStrikeDazeRestriction is what Daze limits; the only value is one of move, action or Bonus Action.
type CunningStrikeDazeRestriction int

// CunningStrikeOneOfMoveActionBonusAction: the target can do only one of move, take an action or take a Bonus Action.
const CunningStrikeOneOfMoveActionBonusAction CunningStrikeDazeRestriction = iota + 1

// CunningStrikeAppliesTo is when a Daze restriction applies; the only value is the target's next turn.
type CunningStrikeAppliesTo int

// CunningStrikeNextTurnOfTarget: the restriction applies on the target's next turn.
const CunningStrikeNextTurnOfTarget CunningStrikeAppliesTo = iota + 1

// CunningStrikeRequiredCover is the cover the Rogue must end the turn behind for Stealth Attack; the only value is
// Three-Quarters Cover or Total Cover.
type CunningStrikeRequiredCover int

// CunningStrikeThreeQuartersOrTotal: Three-Quarters Cover or Total Cover.
const CunningStrikeThreeQuartersOrTotal CunningStrikeRequiredCover = iota + 1

// CunningStrikeDazeOnFailedSave: on a failed saving throw the target, on its next turn, can do only one of move, take an
// action or take a Bonus Action.
type CunningStrikeDazeOnFailedSave struct {
	CunningStrikeCommon
	SaveAbility ability.Ability
	DC          d20.DifficultyClass
	Restriction CunningStrikeDazeRestriction
	AppliesTo   CunningStrikeAppliesTo
}

// CunningStrikeWithdrawMovement: the Rogue moves up to MaxFeet without provoking Opportunity Attacks.
type CunningStrikeWithdrawMovement struct {
	CunningStrikeCommon
	MaxFeet                    int
	ProvokesOpportunityAttacks bool
}

// CunningStrikeStealthAttackKeepsInvisible: the attack does not end the Invisible condition from the Hide action if the
// Rogue ends the turn behind Three-Quarters Cover or Total Cover.
type CunningStrikeStealthAttackKeepsInvisible struct {
	CunningStrikeCommon
	Condition     content.ConditionKind
	RequiredCover CunningStrikeRequiredCover
}

// CunningStrikeNoEffectReason is why a paid effect does nothing; the only value is a target too large for Trip.
type CunningStrikeNoEffectReason int

// CunningStrikeTargetTooLarge: Trip affects a target of size Large or smaller only.
const CunningStrikeTargetTooLarge CunningStrikeNoEffectReason = iota + 1

// CunningStrikeNoEffect: the effect was paid for and does nothing.
type CunningStrikeNoEffect struct {
	CunningStrikeCommon
	Reason CunningStrikeNoEffectReason
}

// Common returns the effect, cost and timing.
func (o CunningStrikeCommon) Common() CunningStrikeCommon { return o }

// CunningStrikeInput is what NewCunningStrikeSelection reads. Construct it with a keyed literal.
type CunningStrikeInput struct {
	// Level is the Rogue level.
	Level RogueLevel
	// Subclass is the Rogue subclass, "" for none.
	Subclass content.SubclassID
	// SneakAttack is the Sneak Attack result of the attack; effects need SneakAttackApplies.
	SneakAttack SneakAttackResult
	// Effects are the chosen effects in order; empty is valid.
	Effects []CunningStrikeEffect
	// PoisonersKit reports a Poisoner's Kit on the character's person.
	PoisonersKit bool
	// DexterityModifier and ProficiencyBonus set the saving throw DC.
	DexterityModifier ability.Modifier
	ProficiencyBonus  ProficiencyBonus
	// TargetSize is the target's size.
	TargetSize content.CreatureSize
	// SpeedFeet is the Rogue's Speed in feet.
	SpeedFeet int
}

// CunningStrikeSelection is a validated choice of Cunning Strike effects. Only NewCunningStrikeSelection builds one; the zero
// value is the empty selection. Immutable; not comparable with ==. Safe for concurrent use.
type CunningStrikeSelection struct {
	outcomes  []CunningStrikeOutcome
	cost      SneakAttackDice
	remaining SneakAttackDice
}

// NewCunningStrikeSelection validates the effects against the Rogue's level, subclass and Sneak Attack dice (SRD 5.2.1,
// "Level 5: Cunning Strike", p. 63) and returns the selection with each effect's outcome. A total cost equal to the dice is
// allowed. It returns an error matching srderr.ErrInvalidInput: a *RogueProblemsError[CunningStrikeProblem] listing every
// problem once, in check order, or a plain invalid-input error for a zero level or a negative Speed. It never panics.
func NewCunningStrikeSelection(in CunningStrikeInput) (CunningStrikeSelection, error) {
	if in.Level.IsZero() || in.SpeedFeet < 0 {
		return CunningStrikeSelection{}, invalidInput("Cunning Strike input with a zero Rogue level or a negative Speed")
	}
	var dice SneakAttackDice
	applies, dealt := in.SneakAttack.(SneakAttackApplies)
	if dealt {
		dice = applies.dice
	}
	var problems []CunningStrikeProblem
	if len(in.Effects) > 0 && !dealt {
		problems = append(problems, CunningStrikeSneakAttackNotDealt)
	}
	limit := 1
	if in.Level.n >= improvedCunningStrikeLevel {
		limit = 2
	}
	if len(in.Effects) > limit {
		problems = append(problems, CunningStrikeTooManyEffects)
	}
	total := 0
	for i, e := range in.Effects {
		if slices.Contains(in.Effects[:i], e) {
			problems = addOnce(problems, CunningStrikeDuplicateEffect)
		}
		if !e.availableTo(in.Level, in.Subclass) {
			problems = addOnce(problems, CunningStrikeEffectNotAvailable)
		}
		total += e.Cost().n
	}
	if dealt && total > dice.n {
		problems = append(problems, CunningStrikeInsufficientDice)
	}
	if slices.Contains(in.Effects, CunningStrikePoison) && !in.PoisonersKit {
		problems = append(problems, CunningStrikePoisonersKitMissing)
	}
	if err := rogueProblems("Cunning Strike selection", problems); err != nil {
		return CunningStrikeSelection{}, err
	}
	sel := CunningStrikeSelection{cost: SneakAttackDice{n: total}, remaining: SneakAttackDice{n: max(dice.n-total, 0)}}
	for _, e := range in.Effects {
		sel.outcomes = append(sel.outcomes, cunningStrikeOutcome(e, in))
	}
	return sel, nil
}

// cunningStrikeOutcome builds the outcome of a validated effect.
func cunningStrikeOutcome(e CunningStrikeEffect, in CunningStrikeInput) CunningStrikeOutcome {
	common := CunningStrikeCommon{Effect: e, Cost: e.Cost(), Timing: CunningStrikeImmediatelyAfterDamage}
	dc := CunningStrikeDC(in.DexterityModifier, in.ProficiencyBonus)
	repeat := CunningStrikeRepeatSave{Ability: ability.Constitution, Trigger: CunningStrikeEndOfEachOfTargetsTurns, EndsOnSuccess: true}
	switch e {
	case CunningStrikePoison:
		return CunningStrikeConditionOnFailedSave{CunningStrikeCommon: common, SaveAbility: ability.Constitution, DC: dc,
			Condition: content.ConditionPoisoned, Duration: CunningStrikeOneMinute, RepeatSave: repeat}
	case CunningStrikeTrip:
		if in.TargetSize > largestTripTarget {
			return CunningStrikeNoEffect{CunningStrikeCommon: common, Reason: CunningStrikeTargetTooLarge}
		}
		return CunningStrikeConditionOnFailedSave{CunningStrikeCommon: common, SaveAbility: ability.Dexterity, DC: dc,
			Condition: content.ConditionProne, Duration: CunningStrikeNoneSpecified, RepeatSave: CunningStrikeNoRepeatSave{}}
	case CunningStrikeWithdraw:
		return CunningStrikeWithdrawMovement{CunningStrikeCommon: common, MaxFeet: in.SpeedFeet / 2}
	case CunningStrikeDaze:
		return CunningStrikeDazeOnFailedSave{CunningStrikeCommon: common, SaveAbility: ability.Constitution, DC: dc,
			Restriction: CunningStrikeOneOfMoveActionBonusAction, AppliesTo: CunningStrikeNextTurnOfTarget}
	case CunningStrikeKnockOut:
		return CunningStrikeConditionOnFailedSave{CunningStrikeCommon: common, SaveAbility: ability.Constitution, DC: dc,
			Condition: content.ConditionUnconscious, Duration: CunningStrikeOneMinute, RepeatSave: repeat, EndsOnDamage: true}
	case CunningStrikeObscure:
		return CunningStrikeConditionOnFailedSave{CunningStrikeCommon: common, SaveAbility: ability.Dexterity, DC: dc,
			Condition: content.ConditionBlinded, Duration: CunningStrikeUntilEndOfTargetsNextTurn, RepeatSave: CunningStrikeNoRepeatSave{}}
	case CunningStrikeStealthAttack:
		return CunningStrikeStealthAttackKeepsInvisible{CunningStrikeCommon: common, Condition: content.ConditionInvisible,
			RequiredCover: CunningStrikeThreeQuartersOrTotal}
	}
	return CunningStrikeNoEffect{CunningStrikeCommon: common} // validation allows no other effect
}

// CunningStrikeDC returns the saving throw DC of a Cunning Strike effect: 8 plus the Dexterity modifier and the Proficiency
// Bonus (SRD 5.2.1, "Level 5: Cunning Strike", p. 63).
func CunningStrikeDC(dexterity ability.Modifier, bonus ProficiencyBonus) d20.DifficultyClass {
	return d20.DifficultyClass(cunningStrikeDCBase + dexterity.Value() + bonus.Value())
}

// Outcomes returns the outcome of each selected effect, in order, as a new slice.
func (s CunningStrikeSelection) Outcomes() []CunningStrikeOutcome {
	return append([]CunningStrikeOutcome{}, s.outcomes...)
}

// Cost returns the total die cost of the selection.
func (s CunningStrikeSelection) Cost() SneakAttackDice { return s.cost }

// RemainingDice returns the Sneak Attack dice left to roll: the dice less the cost (SRD 5.2.1, "Level 5: Cunning Strike",
// p. 63, "You remove the die before rolling"). It is none when Sneak Attack was not dealt.
func (s CunningStrikeSelection) RemainingDice() SneakAttackDice { return s.remaining }
