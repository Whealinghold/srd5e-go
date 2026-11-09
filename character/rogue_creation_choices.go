package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// RogueChoiceProblem is one problem of the choices NewRogueChoices validates: a required choice that is missing, or the
// problem of one of the operations that validate a choice. Kind says which choice; exactly one of Missing and the three
// operation problems is set. Comparable with ==; build it with a keyed literal.
type RogueChoiceProblem struct {
	Kind RogueChoiceKind
	// Missing: the choice is required and none was given.
	Missing bool
	// Skill is the problem of the skill choice, for Kind RogueChoiceSkills.
	Skill RogueSkillChoiceProblem
	// Expertise is the problem of the Expertise selection, for Kind RogueChoiceExpertise.
	Expertise RogueExpertiseProblem
	// Language is the problem of the language choice, for Kind RogueChoiceLanguage.
	Language RogueLanguageChoiceProblem
}

// RogueChoicesInput is what NewRogueChoices reads: the entry and the Rogue level gained, the untrusted picks and the facts
// the rules need. Construct it with a keyed literal. A pick left empty is a missing choice when the level requires it.
type RogueChoicesInput struct {
	Progression RogueProgression
	Entry       RogueEntry
	// Level is the Rogue level gained.
	Level RogueLevel
	// Skills are the skills chosen on entry (level 1).
	Skills []content.Skill
	// Expertise are the two skills chosen for Expertise.
	Expertise []content.Skill
	// Language is the language chosen with Thieves' Cant; the zero Language is none.
	Language content.Language
	// PermittedLanguages are the languages the caller allows.
	PermittedLanguages []content.Language
	// Equipment is the Starting Equipment option; the zero value is none.
	Equipment RogueStartingEquipmentOption
	// Proficient are the character's skill proficiencies before this level gain; the skills chosen at level 1 count too.
	Proficient []content.Skill
	// ExistingExpertise are the skills that already have Expertise.
	ExistingExpertise []content.Skill
	// ExpertiseUsed are the Expertise windows already used.
	ExpertiseUsed []RogueExpertiseWindow
}

// ValidRogueChoices is the accepted choices of a Rogue level gain, as far as the Rogue rules validate them. Immutable; not
// comparable with ==.
type ValidRogueChoices struct {
	skills    RogueSkillChoice
	expertise RogueExpertiseSelection
	language  RogueLanguageChoice
	equipment RogueStartingEquipmentChoice
}

// Skills returns the accepted skill choice, empty when the level required none.
func (c ValidRogueChoices) Skills() RogueSkillChoice { return c.skills }

// Expertise returns the accepted Expertise selection; its window is the zero window when the level required none.
func (c ValidRogueChoices) Expertise() RogueExpertiseSelection { return c.expertise }

// Language returns the accepted language choice; the zero Language when the level required none.
func (c ValidRogueChoices) Language() RogueLanguageChoice { return c.language }

// Equipment returns the accepted Starting Equipment option; zero when the level required none.
func (c ValidRogueChoices) Equipment() RogueStartingEquipmentChoice { return c.equipment }

// New validates the choices the Rogue rules own (skills, Expertise, the Thieves' Cant language, the Starting Equipment option)
// for the entry and level, and only those the level requires. The subclass, a feat and Weapon Mastery appear in RequiredFor
// but are validated by their owners. It returns an error matching srderr.ErrInvalidInput: a
// *RogueProblemsError[RogueChoiceProblem] listing every invalid or missing choice, never only the first, in the order of
// RequiredFor. An undefined entry or a zero level is a plain invalid-input error.
func (c RogueChoices) New(in RogueChoicesInput) (ValidRogueChoices, error) {
	if !in.Entry.Valid() || in.Level.IsZero() || in.Progression.IsZero() {
		return ValidRogueChoices{}, invalidInput("Rogue choices with an undefined entry, a zero level or the zero progression")
	}
	var out ValidRogueChoices
	var problems []RogueChoiceProblem
	proficient := in.Proficient
	for _, kind := range c.RequiredFor(in.Entry, in.Level) {
		switch kind {
		case RogueChoiceSkills:
			if len(in.Skills) == 0 {
				problems = append(problems, RogueChoiceProblem{Kind: RogueChoiceSkills, Missing: true})
				continue
			}
			skills, err := in.Progression.SkillChoice(in.Entry, in.Skills)
			problems = append(problems, wrapProblems(err, func(p RogueSkillChoiceProblem) RogueChoiceProblem {
				return RogueChoiceProblem{Kind: RogueChoiceSkills, Skill: p}
			})...)
			out.skills = skills
			proficient = slices.Concat(proficient, in.Skills)
		case RogueChoiceExpertise:
			if len(in.Expertise) == 0 {
				problems = append(problems, RogueChoiceProblem{Kind: RogueChoiceExpertise, Missing: true})
				continue
			}
			sel, err := RogueExpertise{}.Select(in.Level, proficient, in.ExistingExpertise, in.ExpertiseUsed, in.Expertise)
			problems = append(problems, wrapProblems(err, func(p RogueExpertiseProblem) RogueChoiceProblem {
				return RogueChoiceProblem{Kind: RogueChoiceExpertise, Expertise: p}
			})...)
			out.expertise = sel
		case RogueChoiceLanguage:
			if in.Language == 0 {
				problems = append(problems, RogueChoiceProblem{Kind: RogueChoiceLanguage, Missing: true})
				continue
			}
			lang, err := NewRogueLanguageChoice(in.Language, in.PermittedLanguages)
			problems = append(problems, wrapProblems(err, func(p RogueLanguageChoiceProblem) RogueChoiceProblem {
				return RogueChoiceProblem{Kind: RogueChoiceLanguage, Language: p}
			})...)
			out.language = lang
		case RogueChoiceStartingEquipment:
			eq, err := NewRogueStartingEquipmentChoice(in.Equipment)
			if err != nil {
				problems = append(problems, RogueChoiceProblem{Kind: RogueChoiceStartingEquipment, Missing: true})
			}
			out.equipment = eq
		case RogueChoiceWeaponMastery, RogueChoiceSubclass, RogueChoiceFeat: // validated by their owners
		}
	}
	if err := rogueProblems("Rogue choices", problems); err != nil {
		return ValidRogueChoices{}, err
	}
	return out, nil
}

// wrapProblems maps the problems inside err (a *RogueProblemsError[P]) to RogueChoiceProblems.
func wrapProblems[P comparable](err error, wrap func(P) RogueChoiceProblem) []RogueChoiceProblem {
	var pe *RogueProblemsError[P]
	if !errors.As(err, &pe) {
		return nil
	}
	var out []RogueChoiceProblem
	for _, p := range pe.problems {
		out = append(out, wrap(p))
	}
	return out
}

// RogueCreationChoices is the ClassCreationChoices of a new first-class Rogue: the validated skills, Expertise, language and
// Starting Equipment option of level 1. It seeds the RogueState under RogueStateKey, with the level 1 Expertise window used and
// the language recorded, and names no content. Weapon Mastery is validated and kept by its owner (WeaponMasteryChoiceKey).
// Immutable; not comparable with ==. The zero value is invalid.
type RogueCreationChoices struct{ choices ValidRogueChoices }

// NewRogueCreationChoices validates the level 1 choices of a first-class Rogue, as NewRogueChoices does for (RogueFirstClass,
// level 1), and returns the choices ready for character creation. It returns the same errors.
func NewRogueCreationChoices(in RogueChoicesInput) (RogueCreationChoices, error) {
	in.Entry, in.Level = RogueFirstClass, MustRogueLevel(1)
	v, err := RogueChoices{}.New(in)
	if err != nil {
		return RogueCreationChoices{}, err
	}
	return RogueCreationChoices{choices: v}, nil
}

// Choices returns the validated choices.
func (c RogueCreationChoices) Choices() ValidRogueChoices { return c.choices }

// SeedFeatureStates returns states with the RogueState of a new Rogue stored under RogueStateKey and every other entry
// unchanged; states itself is unchanged.
func (c RogueCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	state := InitialRogueState().WithExpertise(c.choices.expertise)
	state.otherTongue = c.choices.language.language
	next, err := SetFeatureState(states, RogueStateKey(), state)
	if err != nil { // the key is valid, the state is non-nil; only a foreign type under "rogue" fails
		return states
	}
	return next
}

// ContentKeys returns an empty slice: the Rogue choices name no content entry.
func (RogueCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

var _ ClassCreationChoices = RogueCreationChoices{}
