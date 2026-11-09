package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// FeatQualificationFacts is what the prerequisites of a feat read about a
// character. Construct it with a keyed literal. The zero value is a character of
// no level and no features.
type FeatQualificationFacts struct {
	// Level is the total character level ("Level 4+" is total level, SRD 5.2.1, p. 87).
	Level Level
	// Scores are the ability scores before any benefit of the feat.
	Scores ability.Scores
	// Features are the class features held.
	Features HeldFeatures
}

// FeatQualification checks the prerequisites of a feat (SRD 5.2.1, "Parts of a
// Feat", Prerequisite, p. 87). The zero value is ready to use. It is stateless
// and safe for concurrent use.
type FeatQualification struct{}

// Check returns every unmet prerequisite clause of feat, in the feat's clause
// order; the result is empty when the character qualifies, and a feat with no
// clause qualifies every character. The clauses combine with AND; the abilities
// of one ability score clause with OR. A feature clause is met by a held feature
// that table N maps its name to; a name not in the table is never met. It does
// not look at feats already held and never panics.
func (FeatQualification) Check(feat content.Feat, facts FeatQualificationFacts) []content.Prerequisite {
	var unmet []content.Prerequisite
	for _, p := range feat.Prerequisites() {
		if !prerequisiteMet(p, facts) {
			unmet = append(unmet, p)
		}
	}
	return unmet
}

func prerequisiteMet(p content.Prerequisite, facts FeatQualificationFacts) bool {
	switch v := p.(type) {
	case content.LevelPrerequisite:
		return facts.Level.Value() >= v.Minimum()
	case content.AbilityScorePrerequisite:
		return slices.ContainsFunc(v.Abilities(), func(a ability.Ability) bool { return facts.Scores.Score(a).Value() >= v.Minimum() })
	default: // content.FeaturePrerequisite, the last member of the closed set
		clause, _ := p.(content.FeaturePrerequisite)
		return slices.ContainsFunc(featureClauseIDs(clause.Name()), facts.Features.Contains)
	}
}

// featureClauseIDs is table N: the held feature ids that meet a feature clause,
// by exact, case-sensitive name. "Spellcasting Feature" is not met by Pact Magic,
// a separate feature. The id of a class feature is its class, an underscore and
// the slug of its printed name (see content.DeriveFeatureID).
func featureClauseIDs(clause string) []content.FeatureID {
	var owners []string
	var slug string
	switch clause {
	case "Fighting Style Feature":
		owners, slug = []string{"fighter", "paladin", "ranger"}, "fighting_style"
	case "Spellcasting Feature":
		owners, slug = []string{"bard", "cleric", "druid", "paladin", "ranger", "sorcerer", "wizard"}, "spellcasting"
	default:
		return nil
	}
	ids := make([]content.FeatureID, 0, len(owners))
	for _, o := range owners {
		ids = append(ids, content.FeatureID(o+"_"+slug))
	}
	return ids
}
