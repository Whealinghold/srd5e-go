package character

import (
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

const searUndeadLevel = 5

// SearUndeadChoice says whether a Turn Undead also rolls Sear Undead; the SRD says "you can roll",
// so the default (zero) value is to skip. SearUndeadUse asks for the roll.
type SearUndeadChoice int

const (
	// SearUndeadSkip rolls nothing and reports no Sear Undead data.
	SearUndeadSkip SearUndeadChoice = iota
	// SearUndeadUse rolls the Sear Undead dice once, whatever the number of targets.
	SearUndeadUse
)

// UndeadTarget are the host's facts about a creature chosen for Turn Undead. Key is the host's
// opaque key and is never an attribute. Construct it with a keyed literal.
type UndeadTarget struct {
	Key          string
	Type         content.CreatureType
	DistanceFeet int
}

// TurnUndeadEnding is a way the Turn Undead effect ends early on a creature.
type TurnUndeadEnding int

const (
	// TurnUndeadEndsOnDamage: the creature takes any damage (not the Sear Undead damage).
	TurnUndeadEndsOnDamage TurnUndeadEnding = iota + 1
	// TurnUndeadEndsOnClericIncapacitated: the Cleric has the Incapacitated condition.
	TurnUndeadEndsOnClericIncapacitated
	// TurnUndeadEndsOnClericDeath: the Cleric dies.
	TurnUndeadEndsOnClericDeath
)

// String returns "damage", "cleric_incapacitated", "cleric_death" or "unknown".
func (e TurnUndeadEnding) String() string {
	switch e {
	case TurnUndeadEndsOnDamage:
		return "damage"
	case TurnUndeadEndsOnClericIncapacitated:
		return "cleric_incapacitated"
	case TurnUndeadEndsOnClericDeath:
		return "cleric_death"
	}
	return "unknown"
}

// TurnUndead is the pure service of the Turn Undead Channel Divinity effect (SRD 5.2.1, "Level 2:
// Channel Divinity", p. 37). It is stateless; the zero value is ready to use. Safe for concurrent use.
type TurnUndead struct{}

// TargetProblem returns the first target, in command order, that is refused and why:
// target_not_undead, then target_out_of_range (more than 30 feet; exactly 30 is accepted). The
// reason is 0 and the key empty when every target is acceptable. It returns an error matching
// srderr.ErrInvalidInput for a negative distance.
func (TurnUndead) TargetProblem(targets []UndeadTarget) (ClericReason, string, error) {
	for _, t := range targets {
		if t.DistanceFeet < 0 {
			return 0, "", invalidInput("Turn Undead target at a negative distance")
		}
		switch {
		case t.Type != content.CreatureTypeUndead:
			return ClericTargetNotUndead, t.Key, nil
		case t.DistanceFeet > clericEffectRangeFeet:
			return ClericTargetOutOfRange, t.Key, nil
		}
	}
	return 0, "", nil
}

// Conditions returns the conditions a creature that fails its Wisdom saving throw gains for 1 minute:
// Frightened and Incapacitated.
func (TurnUndead) Conditions() []content.ConditionKind {
	return []content.ConditionKind{content.ConditionFrightened, content.ConditionIncapacitated}
}

// EarlyEndings returns the ways the effect ends early: the creature takes any damage, the Cleric has
// the Incapacitated condition, or the Cleric dies.
func (TurnUndead) EarlyEndings() []TurnUndeadEnding {
	return []TurnUndeadEnding{TurnUndeadEndsOnDamage, TurnUndeadEndsOnClericIncapacitated, TurnUndeadEndsOnClericDeath}
}

// SearUndeadDamage is the Radiant damage of a Sear Undead roll, applied to each Undead that fails its
// saving throw. It does not end the Turn Undead effect. Only SearUndead.Resolve builds one.
type SearUndeadDamage struct {
	rolls []int
	total int
}

// Rolls returns the d8s rolled, as a new slice.
func (d SearUndeadDamage) Rolls() []int { return slices.Clone(d.rolls) }

// Total returns the sum of the d8s: the Radiant damage on each failed save.
func (d SearUndeadDamage) Total() int { return d.total }

// DamageType returns Radiant.
func (SearUndeadDamage) DamageType() content.DamageType { return content.DamageRadiant }

// EndsEffect reports whether the damage ends the Turn Undead effect: never.
func (SearUndeadDamage) EndsEffect() bool { return false }

// SearUndead is the pure service of Sear Undead (SRD 5.2.1, "Level 5: Sear Undead", p. 37). It is
// stateless; the zero value is ready to use. Safe for concurrent use.
type SearUndead struct{}

// Dice returns the number of d8s to roll, the Wisdom modifier with a minimum of 1, and false below
// Cleric level 5 or for the zero level.
func (SearUndead) Dice(level ClericLevel, wisdom ability.Modifier) (int, bool) {
	if level.n < searUndeadLevel {
		return 0, false
	}
	return max(wisdom.Value(), 1), true
}

// Resolve adds up the d8s rolled. It returns an error matching srderr.ErrInvalidInput below Cleric
// level 5, for a roll outside 1 to 8 or for a number of rolls other than Dice. It never panics.
func (s SearUndead) Resolve(level ClericLevel, wisdom ability.Modifier, rolls []int) (SearUndeadDamage, error) {
	n, ok := s.Dice(level, wisdom)
	if !ok || len(rolls) != n {
		return SearUndeadDamage{}, invalidInput("Sear Undead: " + strconv.Itoa(len(rolls)) + " rolls or Cleric level below 5")
	}
	total := 0
	for _, r := range rolls {
		if r < 1 || r > 8 {
			return SearUndeadDamage{}, invalidInput("Sear Undead roll " + strconv.Itoa(r) + " outside 1 to 8")
		}
		total += r
	}
	return SearUndeadDamage{rolls: slices.Clone(rolls), total: total}, nil
}
