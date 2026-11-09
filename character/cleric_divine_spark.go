package character

import (
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

const (
	// clericEffectRangeFeet is the range of Divine Spark and Turn Undead and of Preserve Life.
	clericEffectRangeFeet = 30
	// supremeHealingLevel is the Cleric level of Supreme Healing.
	supremeHealingLevel = 17
)

// SparkMode is what a Divine Spark does: restore Hit Points or force a Constitution saving
// throw. The zero value is invalid.
type SparkMode int

const (
	// SparkHealing restores Hit Points equal to the total.
	SparkHealing SparkMode = iota + 1
	// SparkDamage forces a Constitution saving throw; a failed save takes the total as damage.
	SparkDamage
)

// DivineSparkTarget are the host's facts about the creature a Divine Spark is pointed at.
// Key is the host's opaque key and is never an attribute. Construct it with a keyed literal.
type DivineSparkTarget struct {
	Key          string
	IsSelf       bool
	Visible      bool
	DistanceFeet int
}

// DivineSparkResolution is the outcome of a Divine Spark: the d8s rolled, the total (the
// d8s plus the Wisdom modifier, possibly below 0) and the amounts that follow from it
// (SRD 5.2.1, "Level 2: Channel Divinity", p. 37). Only DivineSpark.Resolve builds one.
type DivineSparkResolution struct {
	mode       SparkMode
	rolls      []int
	total      int
	damageType content.DamageType
}

// Mode returns the mode the spark was resolved in.
func (r DivineSparkResolution) Mode() SparkMode { return r.mode }

// Rolls returns the d8s rolled, as a new slice.
func (r DivineSparkResolution) Rolls() []int { return slices.Clone(r.rolls) }

// Total returns the d8s plus the Wisdom modifier; it is below 0 for a low roll and a negative modifier.
func (r DivineSparkResolution) Total() int { return r.total }

// HitPointsToRestore returns the Hit Points to restore in healing mode, never below 0 and uncapped
// by the target's maximum, which is the host's; 0 in damage mode.
func (r DivineSparkResolution) HitPointsToRestore() int {
	if r.mode != SparkHealing {
		return 0
	}
	return max(r.total, 0)
}

// DamageOnFailedSave returns the damage on a failed Constitution save, never below 0; 0 in healing mode.
func (r DivineSparkResolution) DamageOnFailedSave() int {
	if r.mode != SparkDamage {
		return 0
	}
	return max(r.total, 0)
}

// DamageOnSuccessfulSave returns half the failed-save damage, rounded down; 0 in healing mode.
func (r DivineSparkResolution) DamageOnSuccessfulSave() int { return r.DamageOnFailedSave() / 2 }

// DamageType returns the Cleric's choice of Necrotic or Radiant in damage mode; the zero value in healing mode.
func (r DivineSparkResolution) DamageType() content.DamageType { return r.damageType }

// DivineSpark is the pure service of the Divine Spark Channel Divinity effect (SRD 5.2.1,
// "Level 2: Channel Divinity", p. 37). It is stateless; the zero value is ready to use. Safe
// for concurrent use.
type DivineSpark struct{}

// Dice returns the number of d8s rolled at the Cleric level: 1, then 2 from level 7, 3 from
// level 13 and 4 from level 18. The zero level gives 0.
func (DivineSpark) Dice(level ClericLevel) int {
	if level.IsZero() {
		return 0
	}
	return 1 + thresholdCount(level.n, 7, 13, 18)
}

// AllowsDamageType reports whether t is a damage type the Cleric may choose: Necrotic or Radiant.
func (DivineSpark) AllowsDamageType(t content.DamageType) bool {
	return t == content.DamageNecrotic || t == content.DamageRadiant
}

// TargetProblem returns the first reason the target is refused, in the order of class-cleric.md
// FR-128: target_is_self, target_not_visible, target_out_of_range (more than 30 feet; exactly 30
// is accepted); 0 when the target is acceptable. It returns an error matching srderr.ErrInvalidInput
// for a negative distance.
func (DivineSpark) TargetProblem(t DivineSparkTarget) (ClericReason, error) {
	if t.DistanceFeet < 0 {
		return 0, invalidInput("Divine Spark target at a negative distance")
	}
	switch {
	case t.IsSelf:
		return ClericTargetIsSelf, nil
	case !t.Visible:
		return ClericTargetNotVisible, nil
	case t.DistanceFeet > clericEffectRangeFeet:
		return ClericTargetOutOfRange, nil
	}
	return 0, nil
}

// Resolve resolves a Divine Spark of the Cleric level with the d8s rolled and the Wisdom
// modifier. In damage mode damageType must be Necrotic or Radiant.
//
// It returns an error matching srderr.ErrInvalidInput for the zero level, a roll outside 1 to 8,
// a number of rolls other than Dice(level), a mode that is not one of the two, or in damage mode
// another damage type. It never panics.
func (s DivineSpark) Resolve(level ClericLevel, rolls []int, wisdom ability.Modifier, mode SparkMode, damageType content.DamageType) (DivineSparkResolution, error) {
	if level.IsZero() || len(rolls) != s.Dice(level) || mode != SparkHealing && mode != SparkDamage || mode == SparkDamage && !s.AllowsDamageType(damageType) {
		return DivineSparkResolution{}, invalidInput("Divine Spark: " + strconv.Itoa(len(rolls)) + " rolls at Cleric level " + strconv.Itoa(level.n) + " or a damage type that is not allowed")
	}
	total := wisdom.Value()
	for _, r := range rolls {
		if r < 1 || r > 8 {
			return DivineSparkResolution{}, invalidInput("Divine Spark roll " + strconv.Itoa(r) + " outside 1 to 8")
		}
		total += r
	}
	res := DivineSparkResolution{mode: mode, rolls: slices.Clone(rolls), total: total}
	if mode == SparkDamage {
		res.damageType = damageType
	}
	return res, nil
}

// SupremeHealing is the pure service of the Life Domain's Supreme Healing (SRD 5.2.1, "Level 17:
// Supreme Healing", p. 40). It is stateless; the zero value is ready to use. Safe for concurrent use.
type SupremeHealing struct{}

// Applies reports whether healing dice are not rolled: a Cleric of level 17 or higher with the Life
// Domain subclass.
func (SupremeHealing) Applies(level ClericLevel, subclass content.SubclassID) bool {
	return level.n >= supremeHealingLevel && subclass == LifeDomainSubclassID
}

// Maximum returns the highest number possible on count dice of sides faces: 2d6 gives 12.
// It returns an error matching srderr.ErrInvalidInput for fewer than 1 die or fewer than 2 sides.
func (SupremeHealing) Maximum(count, sides int) (int, error) {
	if count < 1 || sides < 2 {
		return 0, invalidInput("healing dice " + strconv.Itoa(count) + "d" + strconv.Itoa(sides) + " need at least one die with two sides")
	}
	return count * sides, nil
}
