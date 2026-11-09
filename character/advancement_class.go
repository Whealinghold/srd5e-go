package character

import "github.com/Whealinghold/srd5e-go/content"

// AdvancementClass is the class a level is gained in, with what LevelUp needs of
// it: its Hit Point Die and its Primary Ability (SRD 5.2.1, "Multiclassing",
// p. 24). The use case builds it from the class definition. It is immutable;
// the zero value is invalid. Build it with NewAdvancementClass.
type AdvancementClass struct {
	class       content.ClassID
	die         content.HitPointDie
	requirement content.AbilityRequirement
}

// NewAdvancementClass returns the class with its Hit Point Die and Primary
// Ability. It returns an error matching srderr.ErrInvalidInput for an invalid
// id, die or a nil requirement. It never panics.
func NewAdvancementClass(class content.ClassID, die content.HitPointDie, requirement content.AbilityRequirement) (AdvancementClass, error) {
	if !class.Valid() || !die.Valid() || requirement == nil || len(requirement.Abilities()) == 0 {
		return AdvancementClass{}, invalidInput("advancement class")
	}
	return AdvancementClass{class: class, die: die, requirement: requirement}, nil
}

// Class returns the class id.
func (a AdvancementClass) Class() content.ClassID { return a.class }

// HitPointDie returns the class's Hit Point Die.
func (a AdvancementClass) HitPointDie() content.HitPointDie { return a.die }

// PrimaryAbility returns the class's Primary Ability requirement.
func (a AdvancementClass) PrimaryAbility() content.AbilityRequirement { return a.requirement }

// IsZero reports whether a is the invalid zero AdvancementClass.
func (a AdvancementClass) IsZero() bool { return a.requirement == nil }
