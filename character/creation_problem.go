package character

import (
	"cmp"
	"math"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// CreationStep is the step of "Create Your Character" a problem belongs to, in
// SRD step order (SRD 5.2.1, p. 19). The zero value is invalid.
type CreationStep int

// The creation steps, in the order problems are reported.
const (
	// CreationStepClass: the class and what it asks (skills, tools, equipment).
	CreationStepClass CreationStep = iota + 1
	// CreationStepBackground: the background and its increases, tool, feat and equipment.
	CreationStepBackground
	// CreationStepSpecies: the species, its size, choices and grants.
	CreationStepSpecies
	// CreationStepLanguages: the chosen languages.
	CreationStepLanguages
	// CreationStepAbilityScores: the ability score method.
	CreationStepAbilityScores
	// CreationStepAlignment: the alignment.
	CreationStepAlignment
	// CreationStepStartingLevel: the starting level and its plan.
	CreationStepStartingLevel
	// CreationStepStartingEquipment: the starting equipment; no problem belongs here, every equipment problem is reported under the step that owns the option.
	CreationStepStartingEquipment
)

// String returns the snake_case name of s or "unknown".
func (s CreationStep) String() string {
	names := [...]string{"unknown", "class", "background", "species", "languages", "ability_scores", "alignment", "starting_level", "starting_equipment"}
	if s < 1 || int(s) >= len(names) {
		return names[0]
	}
	return names[s]
}

// CreateCharacterProblemKind is the kind of a CreateCharacterProblem. The zero
// value is invalid.
type CreateCharacterProblemKind int

// The kinds of problem, in the order a step reports them (SRD 5.2.1, "Create Your
// Character", pp. 19-24). BackgroundIncreaseExceedsMaximum is defensive: no
// command reaches it, because the highest assignable score is 18 and the largest
// increase is 2.
const (
	// ClassSkillCountWrong: the class skills are not exactly the number the class picks.
	ClassSkillCountWrong CreateCharacterProblemKind = iota + 1
	// ClassSkillNotOffered: a class skill is not among the listed skills.
	ClassSkillNotOffered
	// ClassSkillRepeated: a class skill appears twice.
	ClassSkillRepeated
	// ClassToolCountWrong: the chosen class tools are not exactly the number the class picks.
	ClassToolCountWrong
	// ClassToolNotInCategory: a chosen class tool is outside the categories of the class choice.
	ClassToolNotInCategory
	// ClassEquipmentOptionUnknown: the class has no Starting Equipment option with that letter.
	ClassEquipmentOptionUnknown
	// EquipmentPlaceholderUnresolved: an item refers to a tool choice that was not made.
	EquipmentPlaceholderUnresolved
	// BackgroundIncreaseAbilityNotListed: the increase names an ability the background does not list.
	BackgroundIncreaseAbilityNotListed
	// BackgroundIncreaseExceedsMaximum: the increase would raise a score above 20.
	BackgroundIncreaseExceedsMaximum
	// BackgroundToolNotAllowed: a tool was chosen for a specific Tool Proficiency.
	BackgroundToolNotAllowed
	// BackgroundToolMissing: the Tool Proficiency is a choice and no tool was chosen.
	BackgroundToolMissing
	// BackgroundToolNotInCategory: the chosen tool is outside the background's category.
	BackgroundToolNotInCategory
	// BackgroundEquipmentOptionUnknown: the background has no option with that letter.
	BackgroundEquipmentOptionUnknown
	// BackgroundFeatRejected: the background's Origin feat was refused.
	BackgroundFeatRejected
	// SizeRequired: the species has two sizes and none was chosen.
	SizeRequired
	// SizeNotOffered: the chosen size is not one of the species' sizes.
	SizeNotOffered
	// SpeciesChoiceMissing: a species choice has no selection.
	SpeciesChoiceMissing
	// SpeciesChoiceOptionUnknown: a selection names an option the choice lacks.
	SpeciesChoiceOptionUnknown
	// SpeciesChoiceNotOffered: a selection is for a trait the species has no choice for.
	SpeciesChoiceNotOffered
	// SpeciesChoiceRepeated: two selections answer one trait.
	SpeciesChoiceRepeated
	// SpeciesGrantNotOffered: a spellcasting ability is keyed by a trait the species has no choice for.
	SpeciesGrantNotOffered
	// SpeciesFeatNotOrigin: the species-granted feat is not an Origin feat.
	SpeciesFeatNotOrigin
	// SpeciesAbilityNotOffered: the spellcasting ability is not Intelligence, Wisdom or Charisma.
	SpeciesAbilityNotOffered
	// SpeciesFeatRejected: the species-granted Origin feat was refused.
	SpeciesFeatRejected
	// LanguageCountWrong: the chosen languages are not exactly two.
	LanguageCountWrong
	// LanguageNotStandard: a chosen language is not a Standard language.
	LanguageNotStandard
	// LanguageRepeated: a chosen language is Common or appears twice.
	LanguageRepeated
	// AbilityScoreMethodNotPermitted: the method is not in the rule options.
	AbilityScoreMethodNotPermitted
	// AlignmentNotChoosable: the alignment is Unaligned.
	AlignmentNotChoosable
	// LevelAdvancementCountWrong: the plan has not one step per level above 1.
	LevelAdvancementCountWrong
	// LevelAdvancementRejected: a step of the plan was refused.
	LevelAdvancementRejected
)

func createCharacterProblemNames() []string {
	return []string{
		"unknown", "class_skill_count_wrong", "class_skill_not_offered", "class_skill_repeated", "class_tool_count_wrong",
		"class_tool_not_in_category", "class_equipment_option_unknown", "equipment_placeholder_unresolved",
		"background_increase_ability_not_listed", "background_increase_exceeds_maximum", "background_tool_not_allowed",
		"background_tool_missing", "background_tool_not_in_category", "background_equipment_option_unknown",
		"background_feat_rejected", "size_required", "size_not_offered", "species_choice_missing",
		"species_choice_option_unknown", "species_choice_not_offered", "species_choice_repeated",
		"species_grant_not_offered", "species_feat_not_origin", "species_ability_not_offered", "species_feat_rejected",
		"language_count_wrong", "language_not_standard", "language_repeated", "ability_score_method_not_permitted",
		"alignment_not_choosable", "level_advancement_count_wrong", "level_advancement_rejected",
	}
}

// Valid reports whether k is one of the kinds.
func (k CreateCharacterProblemKind) Valid() bool {
	return k >= ClassSkillCountWrong && int(k) < len(createCharacterProblemNames())
}

// String returns the snake_case name of k, which is also its telemetry value, or
// "unknown".
func (k CreateCharacterProblemKind) String() string {
	if !k.Valid() {
		return createCharacterProblemNames()[0]
	}
	return createCharacterProblemNames()[k]
}

// rank orders kinds within a step: declaration order, except that an unresolved
// placeholder, which two steps report, comes last in either.
func (k CreateCharacterProblemKind) rank() int {
	if k == EquipmentPlaceholderUnresolved {
		return math.MaxInt
	}
	return int(k)
}

// CreateCharacterProblem is one reason CreateCharacter refused a command: the step,
// the kind and, where there is one, the offending ability, skill, language, option
// letter, name, the expected and actual counts of a plan, or the rejection a feat
// or the plan met. It carries typed data and no display text. It is immutable. It
// may hold a rejection that holds a slice, so it is not comparable with ==; use
// Equal. Only CreateCharacter builds one; the zero value is invalid.
type CreateCharacterProblem struct {
	step     CreationStep
	kind     CreateCharacterProblemKind
	ability  ability.Ability
	skill    content.Skill
	language content.Language
	letter   EquipmentOptionLetter
	name     string
	expected int
	actual   int
	feat     FeatRejection
	plan     HigherLevelStartRejection
}

// Step returns the creation step the problem belongs to.
func (p CreateCharacterProblem) Step() CreationStep { return p.step }

// Kind returns the kind of the problem.
func (p CreateCharacterProblem) Kind() CreateCharacterProblemKind { return p.kind }

// Ability returns the offending ability, or the zero Ability for none.
func (p CreateCharacterProblem) Ability() ability.Ability { return p.ability }

// Skill returns the offending skill, or the zero Skill for none.
func (p CreateCharacterProblem) Skill() content.Skill { return p.skill }

// Language returns the offending language, or the zero Language for none.
func (p CreateCharacterProblem) Language() content.Language { return p.language }

// Letter returns the offending option letter, or the zero letter for none.
func (p CreateCharacterProblem) Letter() EquipmentOptionLetter { return p.letter }

// Name returns the offending trait, option, size or tool id, or "" for none.
func (p CreateCharacterProblem) Name() string { return p.name }

// Expected returns the number of steps a plan needed (LevelAdvancementCountWrong) or the number of skills or tools a class picks.
func (p CreateCharacterProblem) Expected() int { return p.expected }

// Actual returns the number of steps, skills or tools the command carried.
func (p CreateCharacterProblem) Actual() int { return p.actual }

// FeatRejection returns the rejection of a BackgroundFeatRejected or
// SpeciesFeatRejected problem, or nil.
func (p CreateCharacterProblem) FeatRejection() FeatRejection { return p.feat }

// PlanRejection returns the rejection of a LevelAdvancementRejected problem, or nil.
func (p CreateCharacterProblem) PlanRejection() HigherLevelStartRejection { return p.plan }

// Equal reports whether p and o are the same problem; rejections are compared by
// their String values.
func (p CreateCharacterProblem) Equal(o CreateCharacterProblem) bool {
	return p.key() == o.key()
}

type problemKey struct {
	step                     CreationStep
	kind                     CreateCharacterProblemKind
	ability                  ability.Ability
	skill                    content.Skill
	language                 content.Language
	letter                   EquipmentOptionLetter
	name                     string
	expected, actual         int
	featRejection, planValue string
}

func (p CreateCharacterProblem) key() problemKey {
	k := problemKey{p.step, p.kind, p.ability, p.skill, p.language, p.letter, p.name, p.expected, p.actual, "", ""}
	if p.feat != nil {
		k.featRejection = p.feat.String()
	}
	if p.plan != nil {
		k.planValue = higherLevelStartRejectionValue(p.plan)
	}
	return k
}

// sortProblems orders problems by step and then by kind (FR-005), keeps the order
// of equal ones and drops repeats.
func sortProblems(in []CreateCharacterProblem) []CreateCharacterProblem {
	out := slices.Clone(in)
	slices.SortStableFunc(out, func(a, b CreateCharacterProblem) int {
		return cmp.Or(cmp.Compare(a.step, b.step), cmp.Compare(a.kind.rank(), b.kind.rank()))
	})
	seen := map[problemKey]bool{}
	return slices.DeleteFunc(out, func(p CreateCharacterProblem) bool {
		k := p.key()
		dup := seen[k]
		seen[k] = true
		return dup
	})
}
