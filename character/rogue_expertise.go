package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// The Rogue levels at which Expertise is chosen (SRD 5.2.1, "Level 1: Expertise", p. 61).
const (
	expertiseLevelOne  = 1
	expertiseLevelSix  = 6
	expertisePickCount = 2
)

// RogueExpertiseWindow is one of the two moments a Rogue chooses Expertise: Rogue level 1 and Rogue level 6. The zero value is
// invalid.
type RogueExpertiseWindow int

// The Expertise windows.
const (
	// ExpertiseLevelOne: the selection of Rogue level 1.
	ExpertiseLevelOne RogueExpertiseWindow = iota + 1
	// ExpertiseLevelSix: the selection of Rogue level 6.
	ExpertiseLevelSix
)

// String returns "level_one", "level_six" or "unknown".
func (w RogueExpertiseWindow) String() string {
	switch w {
	case ExpertiseLevelOne:
		return "level_one"
	case ExpertiseLevelSix:
		return "level_six"
	}
	return "unknown"
}

// RogueExpertiseProblem is why an Expertise selection was rejected. The values are in check order. The zero value is not a problem.
type RogueExpertiseProblem int

const (
	// RogueExpertiseNotDueAtThisLevel: the Rogue level is not 1 or 6; reported alone.
	RogueExpertiseNotDueAtThisLevel RogueExpertiseProblem = iota + 1
	// RogueExpertiseAlready: the window was already used.
	RogueExpertiseAlready
	// RogueExpertiseLevelOneNot: the level 6 selection without a recorded level 1 selection.
	RogueExpertiseLevelOneNot
	// RogueExpertiseWrongCount: not exactly two skills.
	RogueExpertiseWrongCount
	// RogueExpertiseDuplicate: the same skill twice in one selection.
	RogueExpertiseDuplicate
	// RogueExpertiseNotProficient: a chosen skill is not a proficiency of the character.
	RogueExpertiseNotProficient
	// RogueExpertiseAlreadyHasExpertise: a chosen skill already has Expertise.
	RogueExpertiseAlreadyHasExpertise
)

// RogueExpertiseSelection is an accepted Expertise selection: the window and the two skills. Immutable; not comparable with ==.
type RogueExpertiseSelection struct {
	window RogueExpertiseWindow
	skills []content.Skill
}

// Window returns the window the selection was made in.
func (s RogueExpertiseSelection) Window() RogueExpertiseWindow { return s.window }

// Skills returns the two skills as a new slice, in the order chosen. The host merges them into SkillProficiencies' Expertise.
func (s RogueExpertiseSelection) Skills() []content.Skill { return slices.Clone(s.skills) }

// RogueExpertise is the Expertise choice rule (SRD 5.2.1, "Level 1: Expertise", p. 61; "Expertise", p. 182). The zero value
// is ready to use.
type RogueExpertise struct{}

// Select validates a selection of two skills at the Rogue level. proficient are the character's skill proficiencies, existing
// the skills that already have Expertise and used the windows already used. Rogue level 1 and Rogue level 6 each allow one
// selection; level 6 needs the level 1 selection recorded. The level is the Rogue level, not the total level.
//
// It returns an error matching srderr.ErrInvalidInput: a *RogueProblemsError[RogueExpertiseProblem] listing every problem
// once, in check order (a level that is not due is reported alone). It never panics. No argument is changed.
func (RogueExpertise) Select(level RogueLevel, proficient, existing []content.Skill, used []RogueExpertiseWindow, chosen []content.Skill) (RogueExpertiseSelection, error) {
	var window RogueExpertiseWindow
	switch level.n {
	case expertiseLevelOne:
		window = ExpertiseLevelOne
	case expertiseLevelSix:
		window = ExpertiseLevelSix
	default:
		return RogueExpertiseSelection{}, rogueProblems("Expertise selection", []RogueExpertiseProblem{RogueExpertiseNotDueAtThisLevel})
	}
	var problems []RogueExpertiseProblem
	if slices.Contains(used, window) {
		problems = append(problems, RogueExpertiseAlready)
	}
	if window == ExpertiseLevelSix && !slices.Contains(used, ExpertiseLevelOne) {
		problems = append(problems, RogueExpertiseLevelOneNot)
	}
	if len(chosen) != expertisePickCount {
		problems = append(problems, RogueExpertiseWrongCount)
	}
	for i, s := range chosen {
		if slices.Contains(chosen[:i], s) {
			problems = addOnce(problems, RogueExpertiseDuplicate)
		}
	}
	for _, s := range chosen {
		if !slices.Contains(proficient, s) {
			problems = addOnce(problems, RogueExpertiseNotProficient)
		}
	}
	for _, s := range chosen {
		if slices.Contains(existing, s) {
			problems = addOnce(problems, RogueExpertiseAlreadyHasExpertise)
		}
	}
	if err := rogueProblems("Expertise selection", problems); err != nil {
		return RogueExpertiseSelection{}, err
	}
	return RogueExpertiseSelection{window: window, skills: slices.Clone(chosen)}, nil
}
