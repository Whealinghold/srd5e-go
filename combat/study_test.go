package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

var studyR combat.StudyRules

// Requirement: doc/specs/hiding-searching-and-senses.md FR-083 (SRD 5.2.1, "Study [Action]", p. 189) - the Study check is Intelligence.
// Path: happy.
func TestStudyAbility(t *testing.T) {
	t.Parallel()
	if studyR.Ability() != ability.Intelligence {
		t.Errorf("Ability = %v", studyR.Ability())
	}
}

// Requirement: per case (doc/specs/hiding-searching-and-senses.md FR-084 to FR-088; SRD 5.2.1, "Study [Action]", Areas of Knowledge, p. 189)
// Path: per case.
func TestStudySubjectsPerSkill(t *testing.T) {
	t.Parallel()
	topics := func(ts ...combat.KnowledgeTopic) []combat.StudySubject {
		var out []combat.StudySubject
		for _, t := range ts {
			out = append(out, combat.StudyTopic{Topic: t})
		}
		return out
	}
	types := func(cs ...content.CreatureType) []combat.StudySubject {
		var out []combat.StudySubject
		for _, c := range cs {
			out = append(out, combat.StudyCreature{Type: c})
		}
		return out
	}
	cases := []struct {
		name, requirement string
		skill             content.Skill
		want              []combat.StudySubject
	}{
		{"Arcana", "FR-084", content.SkillArcana, append(topics(combat.KnowledgeTopicSpells, combat.KnowledgeTopicMagicItems, combat.KnowledgeTopicEldritchSymbols,
			combat.KnowledgeTopicMagicalTraditions, combat.KnowledgeTopicPlanesOfExistence), types(content.CreatureTypeAberration, content.CreatureTypeConstruct,
			content.CreatureTypeElemental, content.CreatureTypeFey, content.CreatureTypeMonstrosity)...)},
		{"History", "FR-085", content.SkillHistory, append(topics(combat.KnowledgeTopicHistoricEvents, combat.KnowledgeTopicHistoricPeople,
			combat.KnowledgeTopicAncientCivilizations, combat.KnowledgeTopicWars), types(content.CreatureTypeGiant, content.CreatureTypeHumanoid)...)},
		{"Investigation has no creature type", "FR-086", content.SkillInvestigation, topics(combat.KnowledgeTopicTraps, combat.KnowledgeTopicCiphers,
			combat.KnowledgeTopicRiddles, combat.KnowledgeTopicGadgetry)},
		{"Nature", "FR-087", content.SkillNature, append(topics(combat.KnowledgeTopicTerrain, combat.KnowledgeTopicFlora, combat.KnowledgeTopicWeather),
			types(content.CreatureTypeBeast, content.CreatureTypeDragon, content.CreatureTypeOoze, content.CreatureTypePlant)...)},
		{"Religion", "FR-088", content.SkillReligion, append(topics(combat.KnowledgeTopicDeities, combat.KnowledgeTopicReligiousHierarchies,
			combat.KnowledgeTopicReligiousRites, combat.KnowledgeTopicHolySymbols, combat.KnowledgeTopicCults),
			types(content.CreatureTypeCelestial, content.CreatureTypeFiend, content.CreatureTypeUndead)...)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := studyR.SubjectsOf(c.skill); !slices.Equal(got, c.want) {
				t.Errorf("SubjectsOf(%v) = %v, want %v [%s]", c.skill, got, c.want, c.requirement)
			}
			for _, s := range c.want {
				if studyR.SkillFor(s) != c.skill {
					t.Errorf("SkillFor(%v) = %v, want %v [%s]", s, studyR.SkillFor(s), c.skill, c.requirement)
				}
			}
			a, ok := studyR.Assess(c.skill).(combat.SuggestedStudySkill)
			if !ok || a.Skill != c.skill || !slices.Equal(a.Subjects(), c.want) {
				t.Errorf("Assess(%v) = %v", c.skill, a)
			}
			if len(c.want) > 0 {
				a.Subjects()[0] = nil
				if a.Subjects()[0] == nil {
					t.Error("Subjects is not a copy [FR-100]")
				}
				studyR.SubjectsOf(c.skill)[0] = nil
				if studyR.SubjectsOf(c.skill)[0] == nil {
					t.Error("SubjectsOf is not a fresh slice [FR-100]")
				}
			}
		})
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-089, FR-090 - all fourteen creature types and all twenty-one topics are covered by
// exactly one Study skill (this fails if a type or topic is added without a row).
// Path: unhappy (boundary).
func TestStudyPartitions(t *testing.T) {
	t.Parallel()
	for ct := content.CreatureTypeAberration; ct.Valid(); ct++ {
		n := 0
		for s := content.SkillAcrobatics; s.Valid(); s++ {
			if slices.Contains(studyR.SubjectsOf(s), combat.StudySubject(combat.StudyCreature{Type: ct})) {
				n++
			}
		}
		if n != 1 || studyR.SkillFor(combat.StudyCreature{Type: ct}) == 0 {
			t.Errorf("creature type %v is covered by %d skills", ct, n)
		}
	}
	topicCount := 0
	for tp := combat.KnowledgeTopicSpells; tp.Valid(); tp++ {
		topicCount++
		n := 0
		for s := content.SkillAcrobatics; s.Valid(); s++ {
			if slices.Contains(studyR.SubjectsOf(s), combat.StudySubject(combat.StudyTopic{Topic: tp})) {
				n++
			}
		}
		if n != 1 || studyR.SkillFor(combat.StudyTopic{Topic: tp}) == 0 {
			t.Errorf("topic %d is covered by %d skills", tp, n)
		}
	}
	if topicCount != 21 {
		t.Errorf("%d valid topics, want 21", topicCount)
	}
	if combat.KnowledgeTopic(0).Valid() || combat.KnowledgeTopic(22).Valid() || !combat.KnowledgeTopicCults.Valid() || combat.KnowledgeTopicCults != 21 {
		t.Error("KnowledgeTopic does not have exactly twenty-one valid values")
	}
	if studyR.SkillFor(nil) != 0 || studyR.SkillFor(combat.StudyTopic{}) != 0 || studyR.SkillFor(combat.StudyCreature{}) != 0 {
		t.Error("an undefined subject has a skill")
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-091 - an Intelligence skill is SuggestedStudySkill; any other skill, and an
// undefined one, is InvalidStudySkill with no subjects.
// Path: unhappy (invalid input).
func TestStudyAssessInvalidSkills(t *testing.T) {
	t.Parallel()
	for _, s := range []content.Skill{content.SkillPerception, content.SkillAthletics, 0} {
		if a := studyR.Assess(s); a != (combat.InvalidStudySkill{Skill: s}) || len(studyR.SubjectsOf(s)) != 0 || studyR.SubjectsOf(s) == nil {
			t.Errorf("Assess(%v) = %v, subjects %#v", s, a, studyR.SubjectsOf(s))
		}
	}
	for _, s := range []content.Skill{content.SkillArcana, content.SkillHistory, content.SkillInvestigation, content.SkillNature, content.SkillReligion} {
		if _, ok := studyR.Assess(s).(combat.SuggestedStudySkill); !ok {
			t.Errorf("Assess(%v) is not suggested", s)
		}
	}
}
