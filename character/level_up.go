package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// multiclassMinimum is the score that meets a Primary Ability when multiclassing
// (SRD 5.2.1, "Multiclassing", "Prerequisites", p. 24).
const multiclassMinimum = 13

// LevelUpRejection is the closed set of reasons LevelUp refuses a level, in
// the order they are checked. Variants: LevelUpMaximumLevel,
// LevelUpInsufficientExperience, LevelUpPrerequisitesNotMet.
type LevelUpRejection interface{ isLevelUpRejection() }

// LevelUpMaximumLevel is the rejection at total level 20.
type LevelUpMaximumLevel struct{}

// LevelUpInsufficientExperience is the rejection when the Experience Points do
// not reach the minimum for the next total level.
type LevelUpInsufficientExperience struct{}

// LevelUpPrerequisitesNotMet is the rejection of a new class whose multiclass
// prerequisites are unmet. Only LevelUp builds one.
type LevelUpPrerequisitesNotMet struct{ short []ability.Ability }

// ShortAbilities returns a new slice of the abilities below 13 that keep a
// requirement unmet, in declaration order without duplicates.
func (r LevelUpPrerequisitesNotMet) ShortAbilities() []ability.Ability { return slices.Clone(r.short) }

func (LevelUpMaximumLevel) isLevelUpRejection()           {}
func (LevelUpInsufficientExperience) isLevelUpRejection() {}
func (LevelUpPrerequisitesNotMet) isLevelUpRejection()    {}

// LevelUp gains one level in class with the Hit Point gain (SRD 5.2.1, "Gaining
// a Level", p. 23; "Multiclassing", pp. 24-25). A class already held is raised
// by one in place; a new class is appended at level 1 when the Primary Ability
// of the new class and of every current class are met (scores of 13 or more),
// read from requirements, which must then hold every current class. The Hit
// Point maximum grows by gain.Amount of the Constitution modifier, one Hit Die
// of the class's size joins the pool, and one LevelGained event follows.
//
// The checks run in this order: invalid arguments (an error), then
// LevelUpMaximumLevel, LevelUpInsufficientExperience and
// LevelUpPrerequisitesNotMet (TransitionRejected values). The receiver is
// unchanged.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or
// class, a nil gain, a rolled gain of another die than the class's, a fixed gain
// other than the die's fixed value, or, for a new class, a requirements map
// lacking a current class (even at level 20). It never panics.
func (c Character) LevelUp(class AdvancementClass, gain HitPointGain, requirements map[content.ClassID]content.AbilityRequirement) (Transition[LevelUpRejection], error) {
	if c.IsZero() || class.IsZero() {
		return nil, invalidInput("level up of a zero character or class")
	}
	if err := checkGain(class, gain); err != nil {
		return nil, err
	}
	isNew := c.classes.LevelIn(class.class) == 0
	if isNew {
		for _, e := range c.classes.entries {
			if requirements[e.Class] == nil {
				return nil, invalidInput("no requirement for current class " + e.Class.String())
			}
		}
	}
	switch {
	case c.TotalLevel().n >= MaxLevel:
		return rejected[LevelUpRejection](LevelUpMaximumLevel{}), nil
	case !c.IsEligibleForNextLevel():
		return rejected[LevelUpRejection](LevelUpInsufficientExperience{}), nil
	}
	if isNew {
		if short := c.shortAbilities(class, requirements); len(short) > 0 {
			return rejected[LevelUpRejection](LevelUpPrerequisitesNotMet{short: short}), nil
		}
	}
	classes, err := c.classes.WithLevelIn(class.class)
	if err != nil { // unreachable: total level is below 20
		return nil, err
	}
	gained := gain.Amount(c.scores.Modifier(ability.Constitution))
	next := c
	next.classes = classes
	next.pool = c.pool.Plus(class.die)
	next.hitPoints = c.hitPoints.Plus(gained)
	return changed[LevelUpRejection](next, LevelGained{
		id: c.id, class: class.class, classLevel: classes.LevelIn(class.class), totalLevel: classes.TotalLevel(),
		hitPointsGained: gained, hitPointMaximum: next.hitPoints, proficiency: ProficiencyBonusFor(classes.TotalLevel()),
	}), nil
}

// checkGain checks that gain is the kind and size the class allows.
func checkGain(class AdvancementClass, gain HitPointGain) error {
	switch g := gain.(type) {
	case RolledHitPointGain:
		if g.die == class.die && g.value >= 1 && g.value <= class.die.Faces() {
			return nil
		}
	case FixedHitPointGain:
		if g.value == class.die.Faces()/2+1 {
			return nil
		}
	}
	return invalidInput("Hit Point gain does not fit the class's Hit Point Die")
}

// shortAbilities returns the abilities below 13 of every unmet requirement: the
// new class's and each current class's.
func (c Character) shortAbilities(class AdvancementClass, requirements map[content.ClassID]content.AbilityRequirement) []ability.Ability {
	var meeting []ability.Ability
	for _, a := range ability.All() {
		if c.scores.Score(a).Value() >= multiclassMinimum {
			meeting = append(meeting, a)
		}
	}
	all := []content.AbilityRequirement{class.requirement}
	for _, e := range c.classes.entries {
		all = append(all, requirements[e.Class])
	}
	var short []ability.Ability
	for _, a := range ability.All() {
		for _, r := range all {
			if !r.IsMetBy(meeting) && slices.Contains(r.Abilities(), a) && !slices.Contains(meeting, a) {
				short = append(short, a)
				break
			}
		}
	}
	return short
}
