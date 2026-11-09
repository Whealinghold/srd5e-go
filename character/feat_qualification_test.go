package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func qualFacts(t *testing.T, level int, str, dex int, held ...content.FeatureID) character.FeatQualificationFacts {
	t.Helper()
	return character.FeatQualificationFacts{Level: character.MustLevel(level), Scores: scoresOf(t, str, dex, 10, 10, 10, 10), Features: must(character.NewHeldFeatures(held...))}
}

// unmetKinds renders the unmet clauses as "level", "score" or "feature" in order.
func unmetKinds(unmet []content.Prerequisite) []string {
	var out []string
	for _, p := range unmet {
		switch p.(type) {
		case content.LevelPrerequisite:
			out = append(out, "level")
		case content.AbilityScorePrerequisite:
			out = append(out, "score")
		case content.FeaturePrerequisite:
			out = append(out, "feature")
		}
	}
	return out
}

// Requirement: per case
// Path: per case.
func TestFeatQualificationCheck(t *testing.T) {
	t.Parallel()
	fs, wiz, pact := content.FeatureID("fighter_fighting_style"), content.FeatureID("wizard_spellcasting"), content.FeatureID("warlock_pact_magic")
	for _, c := range []struct {
		name, requirement, path string
		feat                    content.Feat
		qf                      character.FeatQualificationFacts
		want                    []string
	}{
		{"no prerequisite", featsSpec + ` FR-008, Table P; SRD 5.2.1, "Origin Feats", p. 87`, "happy", alertFeat(), qualFacts(t, 1, 10, 10), nil},
		{"level 4 qualifies for ASI", featsSpec + " FR-009, Table P", "happy", asiFeat(), qualFacts(t, 4, 10, 10), nil},
		{"level 3 fails ASI", featsSpec + " FR-009, Table P", "unhappy (boundary)", asiFeat(), qualFacts(t, 3, 10, 10), []string{"level"}},
		{"Grappler, Strength 13", featsSpec + " FR-010, Table P", "happy", grapplerFeat(), qualFacts(t, 4, 13, 8), nil},
		{"Grappler, Dexterity 13", featsSpec + " FR-010, Table P", "happy", grapplerFeat(), qualFacts(t, 4, 8, 13), nil},
		{"Grappler, both 13", featsSpec + " FR-010", "happy", grapplerFeat(), qualFacts(t, 4, 13, 13), nil},
		{"Grappler, scores low", featsSpec + " FR-010, Table P", "unhappy (boundary)", grapplerFeat(), qualFacts(t, 4, 12, 12), []string{"score"}},
		{"Grappler, level low", featsSpec + " FR-009, Table P", "unhappy (boundary)", grapplerFeat(), qualFacts(t, 3, 13, 13), []string{"level"}},
		{"Grappler, both unmet in clause order", featsSpec + " FR-012, Table P", "unhappy (rule interaction)", grapplerFeat(), qualFacts(t, 3, 12, 12), []string{"level", "score"}},
		{"Epic Boon at 19", featsSpec + " FR-009, Table P", "unhappy (boundary)", boonFeat("boon_of_fate"), qualFacts(t, 19, 10, 10), nil},
		{"Epic Boon at 20", featsSpec + " FR-009, Table P", "unhappy (boundary)", boonFeat("boon_of_fate"), qualFacts(t, 20, 10, 10), nil},
		{"Epic Boon at 18", featsSpec + " FR-009, Table P", "unhappy (boundary)", boonFeat("boon_of_fate"), qualFacts(t, 18, 10, 10), []string{"level"}},
		{"Spell Recall with Spellcasting", featsSpec + " FR-011, Table P", "happy", recallFeat(), qualFacts(t, 19, 10, 10, wiz), nil},
		{"Spell Recall without Spellcasting", featsSpec + " FR-011, Table P", "unhappy (rule interaction)", recallFeat(), qualFacts(t, 19, 10, 10), []string{"feature"}},
		{"Spell Recall at 18", featsSpec + " FR-011, Table P", "unhappy (rule interaction)", recallFeat(), qualFacts(t, 18, 10, 10, wiz), []string{"level"}},
		{"Archery with Fighting Style", featsSpec + " FR-011, Table P", "unhappy (rule interaction)", archeryFeat(), qualFacts(t, 1, 10, 10, fs), nil},
		{"Archery without it", featsSpec + " FR-011, Table P", "unhappy (rule interaction)", archeryFeat(), qualFacts(t, 4, 10, 10), []string{"feature"}},
		{"Paladin Fighting Style", featsSpec + " FR-011, Table N", "happy", archeryFeat(), qualFacts(t, 2, 10, 10, "paladin_fighting_style"), nil},
		{"Ranger Fighting Style", featsSpec + " FR-011, Table N", "happy", archeryFeat(), qualFacts(t, 2, 10, 10, "ranger_fighting_style"), nil},
		{"Bard Spellcasting", featsSpec + " FR-011, Table N", "happy", recallFeat(), qualFacts(t, 19, 10, 10, "bard_spellcasting"), nil},
		{"Cleric Spellcasting", featsSpec + " FR-011, Table N", "happy", recallFeat(), qualFacts(t, 19, 10, 10, "cleric_spellcasting"), nil},
		{"Druid Spellcasting", featsSpec + " FR-011, Table N", "happy", recallFeat(), qualFacts(t, 19, 10, 10, "druid_spellcasting"), nil},
		{"Paladin Spellcasting", featsSpec + " FR-011, Table N", "happy", recallFeat(), qualFacts(t, 19, 10, 10, "paladin_spellcasting"), nil},
		{"Ranger Spellcasting", featsSpec + " FR-011, Table N", "happy", recallFeat(), qualFacts(t, 19, 10, 10, "ranger_spellcasting"), nil},
		{"Sorcerer Spellcasting", featsSpec + " FR-011, Table N", "happy", recallFeat(), qualFacts(t, 19, 10, 10, "sorcerer_spellcasting"), nil},
		{"Pact Magic is not Spellcasting", featsSpec + " FR-011, Table N", "unhappy (rule interaction)", recallFeat(), qualFacts(t, 19, 10, 10, pact), []string{"feature"}},
		{"Additional Fighting Style alone", featsSpec + " FR-011, Table N", "unhappy (boundary)", archeryFeat(), qualFacts(t, 7, 10, 10, "champion_additional_fighting_style"), []string{"feature"}},
		{"a clause name not in table N", featsSpec + " FR-011, Table N", "unhappy (boundary)", featOf("house", content.FeatGeneral, false, featureReq("spellcasting feature")), qualFacts(t, 5, 10, 10, wiz, fs), []string{"feature"}},
		{"another unknown clause name", featsSpec + " FR-011, Table N", "unhappy (boundary)", featOf("house", content.FeatGeneral, false, featureReq("Rage")), qualFacts(t, 5, 10, 10, wiz, fs), []string{"feature"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := unmetKinds(character.FeatQualification{}.Check(c.feat, c.qf))
			if !slices.Equal(got, c.want) {
				t.Errorf("unmet = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-010 - an ability score clause is met by any one of its abilities, and a clause lists its own unmet abilities.
// Path: unhappy (rule interaction).
func TestFeatQualificationReturnsTheUnmetClauses(t *testing.T) {
	t.Parallel()
	unmet := character.FeatQualification{}.Check(grapplerFeat(), qualFacts(t, 3, 12, 12))
	clause, ok := unmet[1].(content.AbilityScorePrerequisite)
	if !ok || clause.Minimum() != 13 || !slices.Equal(clause.Abilities(), []ability.Ability{ability.Strength, ability.Dexterity}) {
		t.Errorf("unmet = %#v", unmet)
	}
	if got := (character.FeatQualification{}).Check(content.Feat{}, character.FeatQualificationFacts{}); len(got) != 0 {
		t.Errorf("a feat with no clause qualifies everyone: %v", got)
	}
}
