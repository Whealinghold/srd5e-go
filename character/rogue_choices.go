package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// RogueChoiceKind names a choice a Rogue level gain requires (SRD 5.2.1, pp. 61-63). The zero value is invalid.
type RogueChoiceKind int

const (
	// RogueChoiceSkills: the skill proficiencies of the class (4 as a first class, 1 as a multiclass).
	RogueChoiceSkills RogueChoiceKind = iota + 1
	// RogueChoiceExpertise: two skills for Expertise.
	RogueChoiceExpertise
	// RogueChoiceLanguage: the language learned with Thieves' Cant.
	RogueChoiceLanguage
	// RogueChoiceWeaponMastery: two kinds of weapon, validated by the Weapon Mastery rules.
	RogueChoiceWeaponMastery
	// RogueChoiceStartingEquipment: Starting Equipment option A or B.
	RogueChoiceStartingEquipment
	// RogueChoiceSubclass: the Rogue subclass, validated by ChooseSubclass.
	RogueChoiceSubclass
	// RogueChoiceFeat: a feat (Ability Score Improvement, an Epic Boon or another qualifying feat), validated by the feat rules.
	RogueChoiceFeat
)

// String returns the kind as snake_case, or "unknown".
func (k RogueChoiceKind) String() string {
	names := [...]string{"skills", "expertise", "language", "weapon_mastery", "starting_equipment", "subclass", "feat"}
	if k < RogueChoiceSkills || k > RogueChoiceFeat {
		return "unknown"
	}
	return names[k-1]
}

// RogueChoices is the Rogue's choice rule (SRD 5.2.1, pp. 61-63). The zero value is ready to use.
type RogueChoices struct{}

// RequiredFor lists the choices a Rogue level gain requires, in the order they are validated; empty where none is due. Choices
// the Rogue rules do not validate (RogueChoiceWeaponMastery, RogueChoiceSubclass, RogueChoiceFeat) are listed and validated by
// their owners. An undefined entry or a zero level lists nothing.
func (RogueChoices) RequiredFor(entry RogueEntry, level RogueLevel) []RogueChoiceKind {
	out := []RogueChoiceKind{}
	switch {
	case !entry.Valid() || level.IsZero():
	case level.n == 1 && entry == RogueFirstClass:
		out = append(out, RogueChoiceSkills, RogueChoiceExpertise, RogueChoiceLanguage, RogueChoiceWeaponMastery, RogueChoiceStartingEquipment)
	case level.n == 1:
		out = append(out, RogueChoiceSkills, RogueChoiceExpertise, RogueChoiceLanguage, RogueChoiceWeaponMastery)
	case level.n == expertiseLevelSix:
		out = append(out, RogueChoiceExpertise)
	case level.n == rogueSubclassLevel:
		out = append(out, RogueChoiceSubclass)
	case level.n == rogueEpicBoonLevel || slices.Contains(rogueFeatLevels(), level.n):
		out = append(out, RogueChoiceFeat)
	}
	return out
}

// RogueSkillChoiceProblem is why a skill choice was rejected. The zero value is not a problem.
type RogueSkillChoiceProblem int

const (
	// RogueSkillWrongCount: not the number of skills the entry grants.
	RogueSkillWrongCount RogueSkillChoiceProblem = iota + 1
	// RogueSkillDuplicate: the same skill twice.
	RogueSkillDuplicate
	// RogueSkillNotInRogueList: a skill that is not on the Rogue's skill list.
	RogueSkillNotInRogueList
)

// RogueSkillChoice is an accepted choice of Rogue skill proficiencies. Immutable; not comparable with ==.
type RogueSkillChoice struct{ skills []content.Skill }

// Skills returns the skills as a new slice, in the order chosen.
func (c RogueSkillChoice) Skills() []content.Skill { return slices.Clone(c.skills) }

// SkillChoice validates the skills chosen on entry: exactly 4 as a first class, exactly 1 as a multiclass, distinct and from
// the Rogue's skill list (SRD 5.2.1, "Core Rogue Traits", "As a Multiclass Character", p. 61). It returns an error matching
// srderr.ErrInvalidInput: a *RogueProblemsError[RogueSkillChoiceProblem] listing every problem once, in check order; an
// undefined entry is a plain invalid-input error.
func (p RogueProgression) SkillChoice(entry RogueEntry, chosen []content.Skill) (RogueSkillChoice, error) {
	picks := p.firstPicks
	switch entry {
	case RogueFirstClass:
	case RogueMulticlass:
		picks = p.multiPicks
	default:
		return RogueSkillChoice{}, invalidInput("Rogue skill choice with an undefined entry")
	}
	var problems []RogueSkillChoiceProblem
	if len(chosen) != picks {
		problems = append(problems, RogueSkillWrongCount)
	}
	for i, s := range chosen {
		if slices.Contains(chosen[:i], s) {
			problems = addOnce(problems, RogueSkillDuplicate)
		}
	}
	for _, s := range chosen {
		if !slices.Contains(p.skills, s) {
			problems = addOnce(problems, RogueSkillNotInRogueList)
		}
	}
	if err := rogueProblems("Rogue skill choice", problems); err != nil {
		return RogueSkillChoice{}, err
	}
	return RogueSkillChoice{skills: slices.Clone(chosen)}, nil
}

// RogueStartingEquipmentOption is the Starting Equipment option a first-class Rogue takes (SRD 5.2.1, "Core Rogue Traits", p.
// 61). The zero value is no choice.
type RogueStartingEquipmentOption int

const (
	// RogueEquipmentOptionA: the equipment of option A and 8 GP.
	RogueEquipmentOptionA RogueStartingEquipmentOption = iota + 1
	// RogueEquipmentOptionB: 100 GP.
	RogueEquipmentOptionB
)

// RogueStartingEquipmentChoice is an accepted Starting Equipment option. Comparable with ==.
type RogueStartingEquipmentChoice struct{ option RogueStartingEquipmentOption }

// Option returns the option chosen.
func (c RogueStartingEquipmentChoice) Option() RogueStartingEquipmentOption { return c.option }

// NewRogueStartingEquipmentChoice accepts option A or B. A missing choice is a failure, never defaulted: it returns an error
// matching srderr.ErrInvalidInput for any other value.
func NewRogueStartingEquipmentChoice(option RogueStartingEquipmentOption) (RogueStartingEquipmentChoice, error) {
	if option != RogueEquipmentOptionA && option != RogueEquipmentOptionB {
		return RogueStartingEquipmentChoice{}, invalidInput("Rogue Starting Equipment choice missing")
	}
	return RogueStartingEquipmentChoice{option: option}, nil
}

// RogueLanguageChoiceProblem is why the language chosen with Thieves' Cant was rejected. The zero value is not a problem.
type RogueLanguageChoiceProblem int

const (
	// RogueLanguageIsThievesCant: the choice is Thieves' Cant itself, not "one other language".
	RogueLanguageIsThievesCant RogueLanguageChoiceProblem = iota + 1
	// RogueLanguageNotPermitted: the language is not in the set the caller permits.
	RogueLanguageNotPermitted
)

// RogueLanguageChoice is an accepted language chosen with Thieves' Cant. Comparable with ==.
type RogueLanguageChoice struct{ language content.Language }

// Language returns the language chosen.
func (c RogueLanguageChoice) Language() content.Language { return c.language }

// NewRogueLanguageChoice validates the one other language a Rogue learns (SRD 5.2.1, "Level 1: Thieves' Cant", p. 62): not
// Thieves' Cant itself, and in permitted (the Standard and Rare Languages the caller allows). A language the character already
// knows is accepted. It returns an error matching srderr.ErrInvalidInput: a *RogueProblemsError[RogueLanguageChoiceProblem].
func NewRogueLanguageChoice(chosen content.Language, permitted []content.Language) (RogueLanguageChoice, error) {
	var problems []RogueLanguageChoiceProblem
	if chosen == content.LanguageThievesCant {
		problems = append(problems, RogueLanguageIsThievesCant)
	}
	if !slices.Contains(permitted, chosen) {
		problems = append(problems, RogueLanguageNotPermitted)
	}
	if err := rogueProblems("Rogue language choice", problems); err != nil {
		return RogueLanguageChoice{}, err
	}
	return RogueLanguageChoice{language: chosen}, nil
}
