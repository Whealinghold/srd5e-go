package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/character-creation.md section 7, FR-099, FR-101 - the signal names, attribute values and every rejection value
// (the snake_case of each CreateCharacterProblemKind) are a public contract: their strings are pinned here and each appears in
// doc/observability/signals.md and, for the attribute values, in doc/rules/observability.md.
// Path: happy.
func TestCreationSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	pins := []struct{ got, want string }{
		{character.SpanCreate, "srd5e.character.create"},
		{character.SpanRollAbilityScores, "srd5e.character.roll_ability_scores"},
		{character.SpanRollLanguages, "srd5e.character.roll_languages"},
		{character.MetricAbilityScoreRoll, "srd5e.character.ability_score_roll"},
		{character.AttrAbilityScoreMethod, "srd5e.character.ability_score_method"},
		{character.ValueOperationCreate, "create"},
		{character.ValueOperationRollAbilityScores, "roll_ability_scores"},
		{character.ValueOperationRollLanguages, "roll_languages"},
		{character.ValueResultCreated, "created"},
		{character.ValueResultDuplicate, "duplicate"},
		{character.ValueAbilityScoreMethodStandardArray, "standard_array"},
		{character.ValueAbilityScoreMethodRandomGeneration, "random_generation"},
		{character.ValueAbilityScoreMethodPointCost, "point_cost"},
	}
	for kind := character.ClassSkillCountWrong; kind.Valid(); kind++ {
		pins = append(pins, struct{ got, want string }{kind.String(), kind.String()})
	}
	for _, p := range pins {
		if p.got != p.want {
			t.Errorf("constant = %q, want %q", p.got, p.want)
		}
		if !strings.Contains(string(catalogue), "`"+p.want+"`") {
			t.Errorf("signals.md does not list %q", p.want)
		}
		if !strings.HasPrefix(p.want, "srd5e.") && !strings.Contains(string(rules), "`"+p.want+"`") {
			t.Errorf("observability.md does not list %q", p.want)
		}
	}
	for _, c := range []struct {
		step character.CreationStep
		want string
	}{{character.CreationStepClass, "class"}, {character.CreationStepBackground, "background"}, {character.CreationStepSpecies, "species"},
		{character.CreationStepLanguages, "languages"}, {character.CreationStepAbilityScores, "ability_scores"}, {character.CreationStepAlignment, "alignment"},
		{character.CreationStepStartingLevel, "starting_level"}, {character.CreationStepStartingEquipment, "starting_equipment"}, {0, "unknown"}, {9, "unknown"}, {99, "unknown"}} {
		if c.step.String() != c.want {
			t.Errorf("CreationStep(%d).String() = %q, want %q", c.step, c.step.String(), c.want)
		}
	}
}
