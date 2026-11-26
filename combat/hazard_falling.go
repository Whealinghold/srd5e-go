package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

const (
	// fallFeetPerDie is the distance of one d6 of falling damage.
	fallFeetPerDie = 10
	// fallMaxDice is the most d6 a fall deals.
	fallMaxDice = 20
)

// FallDistance is the distance a creature falls, in feet, 0 or more (SRD 5.2.1,
// "Falling [Hazard]", p. 182). The host decides that a creature falls and builds
// the distance. The zero value is a fall of 0 feet. Comparable.
type FallDistance struct{ feet int }

// NewFallDistance returns a fall of feet. It returns an error matching
// srderr.ErrInvalidInput for a negative distance. It never panics.
func NewFallDistance(feet int) (FallDistance, error) {
	if feet < 0 {
		return FallDistance{}, invalidInput("fall distance below 0")
	}
	return FallDistance{feet}, nil
}

// MustFallDistance is NewFallDistance for values known to be valid. It panics with
// the error NewFallDistance returns, and only then.
func MustFallDistance(feet int) FallDistance {
	d, err := NewFallDistance(feet)
	if err != nil {
		panic(err)
	}
	return d
}

// Feet returns the distance in feet.
func (d FallDistance) Feet() int { return d.feet }

// Dice returns how many d6 the fall deals: one per full 10 feet, at most 20 (SRD
// 5.2.1, "Round Down", p. 187).
func (d FallDistance) Dice() int { return min(fallMaxDice, d.feet/fallFeetPerDie) }

// LiquidLanding is how a creature that falls into liquid fared on the landing
// check (SRD 5.2.1, "Falling [Hazard]", p. 182). Variants: NoLandingReaction,
// LandingCheckMade.
type LiquidLanding interface{ isLiquidLanding() }

// NoLandingReaction: the creature made no landing check, or could not (it had no
// Reaction). The fall deals full damage.
type NoLandingReaction struct{}

// LandingCheckMade: the creature used its Reaction on the landing check, which
// the host resolved with d20 and reports as Outcome. Construct it with a keyed
// literal.
type LandingCheckMade struct{ Outcome d20.Outcome }

func (NoLandingReaction) isLiquidLanding() {}
func (LandingCheckMade) isLiquidLanding()  {}

// FallSurface is what a creature lands on. Variants: FallOntoGround,
// FallIntoLiquid. A nil FallSurface is FallOntoGround.
type FallSurface interface{ isFallSurface() }

// FallOntoGround: the creature lands on solid ground; the damage is not halved.
type FallOntoGround struct{}

// FallIntoLiquid: the creature lands in liquid and may have made the landing
// check. Construct it with a keyed literal; a nil Landing is NoLandingReaction.
type FallIntoLiquid struct{ Landing LiquidLanding }

func (FallOntoGround) isFallSurface() {}
func (FallIntoLiquid) isFallSurface() {}

// FallLandingOption is one way to make the landing check: an ability and the skill
// used with it.
type FallLandingOption struct {
	Ability ability.Ability
	Skill   content.Skill
}

// FallLandingCheck publishes the landing check of a fall into liquid: DC 15,
// Strength (Athletics) or Dexterity (Acrobatics) (SRD 5.2.1, "Falling [Hazard]",
// p. 182). It does not resolve the check; the host does, with d20. The zero value
// is ready to use.
type FallLandingCheck struct{}

// DC returns the Difficulty Class, 15.
func (FallLandingCheck) DC() d20.DifficultyClass { return d20.DCMedium }

// Options returns a new slice of the two ways to make the check.
func (FallLandingCheck) Options() []FallLandingOption {
	return []FallLandingOption{
		{ability.Strength, content.SkillAthletics},
		{ability.Dexterity, content.SkillAcrobatics},
	}
}

// fallAdjustments is the damage adjustment a landing earns: Half Damage when the
// creature landed in liquid after a successful landing check.
func fallAdjustments(surface FallSurface) []DamageAdjustment {
	if l, ok := surface.(FallIntoLiquid); ok {
		if c, ok := l.Landing.(LandingCheckMade); ok && c.Outcome == d20.Success {
			return []DamageAdjustment{HalfDamage{}}
		}
	}
	return nil
}

// fallExpression is the roll of a fall of n d6.
func fallExpression(n int) dice.Expression { return dice.MustExpression(n, dice.D6(), 0) }
