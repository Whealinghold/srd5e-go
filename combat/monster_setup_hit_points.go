package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// MonsterHitPointsMethod is how the Hit Points of one monster are decided: the printed
// average or a roll of its Hit Point Dice, never both (SRD 5.2.1, "Hit Points", p. 255).
// Variants: UseAverageHitPoints, RollHitPointDice.
type MonsterHitPointsMethod interface{ isMonsterHitPointsMethod() }

// UseAverageHitPoints takes the printed average as the Hit Point maximum.
type UseAverageHitPoints struct{}

// RollHitPointDice rolls the printed Hit Point Dice, with the printed flat modifier.
type RollHitPointDice struct{}

func (UseAverageHitPoints) isMonsterHitPointsMethod() {}
func (RollHitPointDice) isMonsterHitPointsMethod()    {}

// ResolvedMonsterHitPoints is a decided choice of the Hit Points of one monster.
// Variants: PrintedAverageHitPoints, RolledMonsterHitPoints. No member combines the
// average and a roll.
type ResolvedMonsterHitPoints interface{ isResolvedMonsterHitPoints() }

// PrintedAverageHitPoints is the printed average, used as printed and not recomputed
// from the dice or the Constitution modifier.
type PrintedAverageHitPoints struct{}

// RolledMonsterHitPoints is a roll of the monster's Hit Point Dice. Construct it with a
// keyed literal. Its Roll must be a roll of the printed Hit Point Dice expression,
// including the flat modifier, or NewMonsterSetup reports RolledExpressionMismatch.
type RolledMonsterHitPoints struct{ Roll dice.Result }

func (PrintedAverageHitPoints) isResolvedMonsterHitPoints() {}
func (RolledMonsterHitPoints) isResolvedMonsterHitPoints()  {}

// Equal reports whether r and o are the same roll.
func (r RolledMonsterHitPoints) Equal(o RolledMonsterHitPoints) bool { return r.Roll.Equal(o.Roll) }

// monsterMaximum returns the Hit Point maximum the choice gives a monster with the
// printed entry, and whether a rolled total was raised to 1. A rolled total of 0 or
// less is raised to 1 (interpretation: a creature set up for a fight starts alive; SRD
// 5.2.1, "Hit Point Maximum of 0", p. 17). The problem is RolledExpressionMismatch for
// a roll of another expression, SetupArgumentInvalid for a nil or unknown choice and 0
// when there is none.
func monsterMaximum(entry content.HitPointsEntry, hp ResolvedMonsterHitPoints) (maximum int, raised bool, problem MonsterSetupProblem) {
	switch v := hp.(type) {
	case PrintedAverageHitPoints:
		return entry.Average(), false, 0
	case RolledMonsterHitPoints:
		if v.Roll.Expression() != entry.Dice() {
			return 0, false, RolledExpressionMismatch
		}
		if total := v.Roll.Total(); total >= 1 {
			return total, false, 0
		}
		return 1, true, 0
	}
	return 0, false, SetupArgumentInvalid
}
