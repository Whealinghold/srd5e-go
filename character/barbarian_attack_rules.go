package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/modifier"
)

// BrutalStrikeEffect is one effect a Brutal Strike can cause, a closed set (SRD 5.2.1, "Level 9:
// Brutal Strike", pp. 29-30, "Level 13: Improved Brutal Strike", p. 30). The zero value is not an
// effect.
type BrutalStrikeEffect int

// The Brutal Strike effects.
const (
	// ForcefulBlow: the target is pushed 15 feet straight away; the Barbarian may then move up to half its Speed straight toward it without provoking Opportunity Attacks.
	ForcefulBlow BrutalStrikeEffect = iota + 1
	// HamstringBlow: the target's Speed is reduced by 15 feet until the start of the Barbarian's next turn; only the most recent Hamstring Blow applies.
	HamstringBlow
	// StaggeringBlow: the target has Disadvantage on the next saving throw it makes and can't make Opportunity Attacks until the start of the Barbarian's next turn.
	StaggeringBlow
	// SunderingBlow: the next attack roll by another creature against the target, before the start of the Barbarian's next turn, gains a +5 bonus; an attack roll gains only one Sundering Blow bonus.
	SunderingBlow
)

// The numbers of the Brutal Strike effects, in feet or as a bonus.
const (
	ForcefulBlowPushFeet       = 15
	HamstringBlowSpeedFeet     = 15
	SunderingBlowAttackBonus   = 5
	brutalStrikeBaseDiceCount  = 1
	brutalStrikeImprovedDice   = 2
	brutalStrikeSides          = 10
	frenzySides                = 6
	brutalStrikeOneEffectLimit = 1
	brutalStrikeTwoEffectLimit = 2
)

// String returns "forceful_blow", "hamstring_blow", "staggering_blow", "sundering_blow" or "unknown".
func (e BrutalStrikeEffect) String() string {
	names := []string{"forceful_blow", "hamstring_blow", "staggering_blow", "sundering_blow"}
	if e < 1 || int(e) > len(names) {
		return "unknown"
	}
	return names[e-1]
}

// Valid reports whether e is one of the four effects.
func (e BrutalStrikeEffect) Valid() bool { return e >= ForcefulBlow && e <= SunderingBlow }

// UnlockLevel returns the Barbarian level that gives the effect: 9 for Forceful and Hamstring
// Blow, 13 for Staggering and Sundering Blow; 0 for an invalid value.
func (e BrutalStrikeEffect) UnlockLevel() int {
	switch e {
	case ForcefulBlow, HamstringBlow:
		return brutalStrikeLevel
	case StaggeringBlow, SunderingBlow:
		return improvedBrutalStrike13
	}
	return 0
}

// ForcefulBlowMove returns how far the Barbarian may move straight toward the target after a
// Forceful Blow: half speed in feet, rounded down (SRD 5.2.1, "Round Down", p. 5). It returns an
// error matching srderr.ErrInvalidInput for a negative speed.
func ForcefulBlowMove(speed int) (int, error) {
	if speed < 0 {
		return 0, invalidInput("negative Speed")
	}
	return speed / 2, nil
}

// RecklessOutcome is the closed outcome of BarbarianAttackRules.DeclareReckless. Variants:
// RecklessDeclared, RecklessRefused.
type RecklessOutcome interface{ isRecklessOutcome() }

// RecklessDeclared is Reckless Attack declared. Only BarbarianAttackRules.DeclareReckless builds one.
type RecklessDeclared struct {
	turn      BarbarianTurnState
	advantage bool
}

// Turn returns the new turn state, with Reckless Attack declared.
func (r RecklessDeclared) Turn() BarbarianTurnState { return r.turn }

// AdvantageOnThisAttack reports Advantage on the attack roll being made: true when it uses
// Strength. Attack rolls against the Barbarian have Advantage until the start of its next turn
// whatever the attack uses.
func (r RecklessDeclared) AdvantageOnThisAttack() bool { return r.advantage }

// RecklessRefused is a Reckless Attack a rule refused: nothing changes. Construct it with a keyed literal.
type RecklessRefused struct{ Reason BarbarianReason }

func (RecklessDeclared) isRecklessOutcome() {}
func (RecklessRefused) isRecklessOutcome()  {}

// BrutalStrikeDeclaration is the closed outcome of BarbarianAttackRules.DeclareBrutalStrike. Variants:
// BrutalStrikeDeclared, BrutalStrikeRefused.
type BrutalStrikeDeclaration interface{ isBrutalStrikeDeclaration() }

// BrutalStrikeDeclared is a Brutal Strike chosen before the attack roll. Only
// BarbarianAttackRules.DeclareBrutalStrike builds one.
type BrutalStrikeDeclared struct {
	allowed []BrutalStrikeEffect
	maximum int
}

// ForgoAdvantage is always true: the caller drops every Advantage source on this roll.
func (BrutalStrikeDeclared) ForgoAdvantage() bool { return true }

// AllowedEffects returns the effects the level allows, as a new slice.
func (r BrutalStrikeDeclared) AllowedEffects() []BrutalStrikeEffect {
	return append([]BrutalStrikeEffect{}, r.allowed...)
}

// MaximumEffects returns how many different effects the level allows: 1, or 2 from level 17.
func (r BrutalStrikeDeclared) MaximumEffects() int { return r.maximum }

// BrutalStrikeResolution is the closed outcome of BarbarianAttackRules.ResolveBrutalStrike. Variants:
// BrutalStrikeResolved, BrutalStrikeRefused.
type BrutalStrikeResolution interface{ isBrutalStrikeResolution() }

// BrutalStrikeResolved is a Brutal Strike after the attack roll. Only
// BarbarianAttackRules.ResolveBrutalStrike builds one.
type BrutalStrikeResolved struct {
	extra   BarbarianExtraDamage
	hit     bool
	effects []BrutalStrikeEffect
}

// ExtraDamage returns the extra damage to roll and true when the attack hit; false and the zero
// value on a miss. The dice are returned, not rolled.
func (r BrutalStrikeResolved) ExtraDamage() (BarbarianExtraDamage, bool) { return r.extra, r.hit }

// Effects returns the effects caused, as a new slice: the effects named when the attack hit, none on a miss.
func (r BrutalStrikeResolved) Effects() []BrutalStrikeEffect {
	return append([]BrutalStrikeEffect{}, r.effects...)
}

// BrutalStrikeRefused is a Brutal Strike a rule refused. Construct it with a keyed literal.
type BrutalStrikeRefused struct{ Reason BarbarianReason }

func (BrutalStrikeDeclared) isBrutalStrikeDeclaration() {}
func (BrutalStrikeRefused) isBrutalStrikeDeclaration()  {}
func (BrutalStrikeResolved) isBrutalStrikeResolution()  {}
func (BrutalStrikeRefused) isBrutalStrikeResolution()   {}

// BarbarianAttackRules decides Reckless Attack and Brutal Strike (SRD 5.2.1, "Level 2: Reckless
// Attack", p. 29, "Level 9: Brutal Strike", pp. 29-30, "Level 17: Improved Brutal Strike", p. 30). It
// rolls no dice. It is stateless and pure; the zero value is ready to use. Safe for concurrent use.
type BarbarianAttackRules struct{}

// DeclareReckless declares Reckless Attack on the attack roll attack. The first reason that holds
// refuses it, in this order: BarbarianNotABarbarian, BarbarianFeatureNotGained (below level 2),
// BarbarianNotFirstAttackRoll. An attack that does not use Strength is accepted but gets no
// Advantage (AdvantageOnThisAttack is false).
func (BarbarianAttackRules) DeclareReckless(stats BarbarianLevelStats, turn BarbarianTurnState, attack BarbarianAttackFacts) RecklessOutcome {
	switch {
	case stats.IsZero():
		return RecklessRefused{Reason: BarbarianNotABarbarian}
	case stats.level.n < recklessAttackLevel:
		return RecklessRefused{Reason: BarbarianFeatureNotGained}
	case turn.firstAttackRollMade:
		return RecklessRefused{Reason: BarbarianNotFirstAttackRoll}
	}
	return RecklessDeclared{turn: turn.DeclareReckless(), advantage: attack.UsesStrength}
}

// brutalStrikeLimit returns how many different effects the level allows.
func brutalStrikeLimit(level int) int {
	if level >= improvedBrutalStrike17 {
		return brutalStrikeTwoEffectLimit
	}
	return brutalStrikeOneEffectLimit
}

// DeclareBrutalStrike chooses a Brutal Strike before the attack roll, forgoing every Advantage on
// it. The first reason that holds refuses it, in this order: BarbarianNotABarbarian,
// BarbarianFeatureNotGained (below level 9), BarbarianRecklessNotDeclared,
// BarbarianNotAStrengthAttack.
func (BarbarianAttackRules) DeclareBrutalStrike(stats BarbarianLevelStats, turn BarbarianTurnState, attack BarbarianAttackFacts) BrutalStrikeDeclaration {
	switch {
	case stats.IsZero():
		return BrutalStrikeRefused{Reason: BarbarianNotABarbarian}
	case stats.level.n < brutalStrikeLevel:
		return BrutalStrikeRefused{Reason: BarbarianFeatureNotGained}
	case !turn.recklessDeclared:
		return BrutalStrikeRefused{Reason: BarbarianRecklessNotDeclared}
	case !attack.UsesStrength:
		return BrutalStrikeRefused{Reason: BarbarianNotAStrengthAttack}
	}
	allowed := []BrutalStrikeEffect{ForcefulBlow, HamstringBlow}
	if stats.level.n >= improvedBrutalStrike13 {
		allowed = append(allowed, StaggeringBlow, SunderingBlow)
	}
	return BrutalStrikeDeclared{allowed: allowed, maximum: brutalStrikeLimit(stats.level.n)}
}

// ResolveBrutalStrike resolves a Brutal Strike after the attack roll, whose RollMode attack
// carries after the Advantage was forgone: an attack roll with both Advantage and Disadvantage has
// neither and arrives Normal (SRD 5.2.1, "They Don't Stack", p. 8). The first reason that holds
// refuses it, in this order: BarbarianNotABarbarian, BarbarianFeatureNotGained,
// BarbarianNotAStrengthAttack, BarbarianAttackHasDisadvantage and, when the attack hit,
// BarbarianEffectNotUnlocked, BarbarianDuplicateEffect, BarbarianTooManyEffects. A miss is accepted:
// no extra damage and no effect. A hit gives 1d10 extra damage (2d10 from level 17) of the attack's
// damage type, not rolled, and the effects named: zero effects is valid, up to the limit of the level.
//
// It returns an error matching srderr.ErrInvalidInput for an effect that is not one of the four,
// reported even on a miss. It never panics.
func (BarbarianAttackRules) ResolveBrutalStrike(stats BarbarianLevelStats, attack BarbarianAttackFacts, effects []BrutalStrikeEffect) (BrutalStrikeResolution, error) {
	if slices.ContainsFunc(effects, func(e BrutalStrikeEffect) bool { return !e.Valid() }) {
		return nil, invalidInput("Brutal Strike with an unknown effect")
	}
	level := stats.level.n
	switch {
	case stats.IsZero():
		return BrutalStrikeRefused{Reason: BarbarianNotABarbarian}, nil
	case level < brutalStrikeLevel:
		return BrutalStrikeRefused{Reason: BarbarianFeatureNotGained}, nil
	case !attack.UsesStrength:
		return BrutalStrikeRefused{Reason: BarbarianNotAStrengthAttack}, nil
	case attack.RollMode == modifier.WithDisadvantage:
		return BrutalStrikeRefused{Reason: BarbarianAttackHasDisadvantage}, nil
	case !attack.Hit:
		return BrutalStrikeResolved{effects: []BrutalStrikeEffect{}}, nil
	case slices.ContainsFunc(effects, func(e BrutalStrikeEffect) bool { return e.UnlockLevel() > level }):
		return BrutalStrikeRefused{Reason: BarbarianEffectNotUnlocked}, nil
	case len(slices.Compact(slices.Sorted(slices.Values(effects)))) != len(effects):
		return BrutalStrikeRefused{Reason: BarbarianDuplicateEffect}, nil
	case len(effects) > brutalStrikeLimit(level):
		return BrutalStrikeRefused{Reason: BarbarianTooManyEffects}, nil
	}
	count := brutalStrikeBaseDiceCount
	if level >= improvedBrutalStrike17 {
		count = brutalStrikeImprovedDice
	}
	return BrutalStrikeResolved{
		extra: BarbarianExtraDamage{count: count, die: brutalStrikeSides, damageType: attack.DamageType}, hit: true,
		effects: append([]BrutalStrikeEffect{}, effects...),
	}, nil
}
