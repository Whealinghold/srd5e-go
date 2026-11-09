package character

import "github.com/Whealinghold/srd5e-go/content"

// calligraphersSupplies is the id of Calligrapher's Supplies (SRD 5.2.1,
// "Artisan's Tools", p. 93), the tool that qualifies a scribe beside Arcana.
const calligraphersSupplies content.ItemID = "calligraphers_supplies"

// ScribeQualification is the closed set of outcomes of [QualifyScribe]. Variants:
// ScribeQualified, ScribeNotQualified.
type ScribeQualification interface{ isScribeQualification() }

// ScribeQualified says the scribe has the proficiency to scribe Spell Scrolls.
type ScribeQualified struct{}

// ScribeNotQualified says the scribe has neither Arcana nor Calligrapher's Supplies.
type ScribeNotQualified struct{}

func (ScribeQualified) isScribeQualification()    {}
func (ScribeNotQualified) isScribeQualification() {}

// QualifyScribe says whether a character may scribe Spell Scrolls: proficiency
// in the Arcana skill (Expertise implies it) or with Calligrapher's Supplies;
// either one suffices (SRD 5.2.1, "Prerequisites for the Scribe", p. 103). Being
// a spellcaster is enforced through the prepared-spell prerequisite of
// [ScrollScribingJob.WorkDay]. It is pure and returns no error.
func QualifyScribe(skills SkillProficiencies, tools ToolProficiencies) ScribeQualification {
	if skills.Proficient(content.SkillArcana) || skills.HasExpertise(content.SkillArcana) || tools.Contains(calligraphersSupplies) {
		return ScribeQualified{}
	}
	return ScribeNotQualified{}
}
