package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// KnowledgeTopic is a topic of the Areas of Knowledge table. The zero value is
// invalid. The table prints "historic events and people" and "religious hierarchies
// and rites"; each is split in two so that every topic has one skill.
type KnowledgeTopic int

// The twenty-one topics (SRD 5.2.1, "Study [Action]", Areas of Knowledge, p. 189).
const (
	KnowledgeTopicSpells KnowledgeTopic = iota + 1
	KnowledgeTopicMagicItems
	KnowledgeTopicEldritchSymbols
	KnowledgeTopicMagicalTraditions
	KnowledgeTopicPlanesOfExistence
	KnowledgeTopicHistoricEvents
	KnowledgeTopicHistoricPeople
	KnowledgeTopicAncientCivilizations
	KnowledgeTopicWars
	KnowledgeTopicTraps
	KnowledgeTopicCiphers
	KnowledgeTopicRiddles
	KnowledgeTopicGadgetry
	KnowledgeTopicTerrain
	KnowledgeTopicFlora
	KnowledgeTopicWeather
	KnowledgeTopicDeities
	KnowledgeTopicReligiousHierarchies
	KnowledgeTopicReligiousRites
	KnowledgeTopicHolySymbols
	KnowledgeTopicCults
)

// Valid reports whether t is one of the twenty-one topics.
func (t KnowledgeTopic) Valid() bool { return t >= KnowledgeTopicSpells && t <= KnowledgeTopicCults }

// StudySubject is the closed set of things a Study can be about. Variants:
// StudyTopic, StudyCreature.
type StudySubject interface{ isStudySubject() }

// StudyTopic: a topic of the Areas of Knowledge table.
type StudyTopic struct{ Topic KnowledgeTopic }

// StudyCreature: a creature type of the Areas of Knowledge table.
type StudyCreature struct{ Type content.CreatureType }

func (StudyTopic) isStudySubject()    {}
func (StudyCreature) isStudySubject() {}

// StudySkillAssessment is the closed set of answers of StudyRules.Assess.
// Variants: SuggestedStudySkill, InvalidStudySkill.
type StudySkillAssessment interface{ isStudySkillAssessment() }

// SuggestedStudySkill: an Intelligence skill, with the subjects the table gives it.
type SuggestedStudySkill struct {
	Skill    content.Skill
	subjects []StudySubject
}

// Subjects returns a copy of the subjects of the skill.
func (s SuggestedStudySkill) Subjects() []StudySubject { return slices.Clone(s.subjects) }

// InvalidStudySkill: the skill is not an Intelligence skill; it has no subjects.
type InvalidStudySkill struct{ Skill content.Skill }

func (SuggestedStudySkill) isStudySkillAssessment() {}
func (InvalidStudySkill) isStudySkillAssessment()   {}

// StudyRules is the stateless domain service for the Study action (SRD 5.2.1, "Study
// [Action]", Areas of Knowledge, p. 189). Pure, synchronous and silent; the zero value
// is ready to use. Safe for concurrent use.
type StudyRules struct{}

// Ability returns Intelligence, the ability of a Study check.
func (StudyRules) Ability() ability.Ability { return ability.Intelligence }

// studyAreas lists, per Intelligence skill, its topics and creature types. A function,
// not a package variable (A4).
func studyAreas() []struct {
	skill  content.Skill
	topics []KnowledgeTopic
	types  []content.CreatureType
} {
	return []struct {
		skill  content.Skill
		topics []KnowledgeTopic
		types  []content.CreatureType
	}{
		{content.SkillArcana, []KnowledgeTopic{KnowledgeTopicSpells, KnowledgeTopicMagicItems, KnowledgeTopicEldritchSymbols,
			KnowledgeTopicMagicalTraditions, KnowledgeTopicPlanesOfExistence},
			[]content.CreatureType{content.CreatureTypeAberration, content.CreatureTypeConstruct, content.CreatureTypeElemental,
				content.CreatureTypeFey, content.CreatureTypeMonstrosity}},
		{content.SkillHistory, []KnowledgeTopic{KnowledgeTopicHistoricEvents, KnowledgeTopicHistoricPeople,
			KnowledgeTopicAncientCivilizations, KnowledgeTopicWars},
			[]content.CreatureType{content.CreatureTypeGiant, content.CreatureTypeHumanoid}},
		{content.SkillInvestigation, []KnowledgeTopic{KnowledgeTopicTraps, KnowledgeTopicCiphers, KnowledgeTopicRiddles,
			KnowledgeTopicGadgetry}, nil},
		{content.SkillNature, []KnowledgeTopic{KnowledgeTopicTerrain, KnowledgeTopicFlora, KnowledgeTopicWeather},
			[]content.CreatureType{content.CreatureTypeBeast, content.CreatureTypeDragon, content.CreatureTypeOoze,
				content.CreatureTypePlant}},
		{content.SkillReligion, []KnowledgeTopic{KnowledgeTopicDeities, KnowledgeTopicReligiousHierarchies,
			KnowledgeTopicReligiousRites, KnowledgeTopicHolySymbols, KnowledgeTopicCults},
			[]content.CreatureType{content.CreatureTypeCelestial, content.CreatureTypeFiend, content.CreatureTypeUndead}},
	}
}

// SkillFor returns the one Study skill that covers the subject: Arcana, History,
// Investigation, Nature or Religion. It returns the zero Skill for an undefined or
// nil subject.
func (StudyRules) SkillFor(subject StudySubject) content.Skill {
	for _, a := range studyAreas() {
		switch s := subject.(type) {
		case StudyTopic:
			if slices.Contains(a.topics, s.Topic) {
				return a.skill
			}
		case StudyCreature:
			if slices.Contains(a.types, s.Type) {
				return a.skill
			}
		}
	}
	return 0
}

// SubjectsOf returns a new slice of the topics then the creature types the skill
// covers; it is empty for a skill that is not an Intelligence skill.
func (StudyRules) SubjectsOf(skill content.Skill) []StudySubject {
	out := []StudySubject{}
	for _, a := range studyAreas() {
		if a.skill != skill {
			continue
		}
		for _, t := range a.topics {
			out = append(out, StudyTopic{t})
		}
		for _, c := range a.types {
			out = append(out, StudyCreature{c})
		}
	}
	return out
}

// Assess classifies a skill for a Study: SuggestedStudySkill with its subjects for
// an Intelligence skill, InvalidStudySkill for any other.
func (r StudyRules) Assess(skill content.Skill) StudySkillAssessment {
	if skill.Ability() != r.Ability() {
		return InvalidStudySkill{skill}
	}
	return SuggestedStudySkill{skill, r.SubjectsOf(skill)}
}
