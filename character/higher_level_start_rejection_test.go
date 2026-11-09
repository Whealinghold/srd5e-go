package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-078, FR-123 - the starting arguments the level 1
// Character refuses come back as the rejection's problems, in the order New reports them, as a copy, and a
// score above 20 names its abilities; a well-formed start has none.
// Path: unhappy (invalid input).
func TestHigherLevelStartCreationInvalidCarriesEveryProblem(t *testing.T) {
	t.Parallel()
	cmd := startCmd(t, 3, 21, fighterSteps(2, false, nil))
	cmd.AbilityScores = scoresOf(t, 21, 10, 14, 10, 22, 10)
	reason := rejectionOf(t, startOf(t, startUseCase(t, hlsSource(t, hlsFighter(t)), dicetest.NewMinimumRoller()), cmd))
	invalid, ok := reason.(character.HigherLevelStartCreationInvalid)
	if !ok || len(invalid.Problems()) != 1 {
		t.Fatalf("rejection = %#v, want one creation problem", reason)
	}
	above, ok := invalid.Problems()[0].(character.StartingScoreAboveMaximum)
	if !ok || !reflect.DeepEqual(above.Abilities(), []ability.Ability{ability.Strength, ability.Wisdom}) {
		t.Errorf("problem = %#v, want StartingScoreAboveMaximum [Strength Wisdom]", invalid.Problems()[0])
	}
	if got := (character.HigherLevelStartCreationInvalid{}).Problems(); got != nil {
		t.Errorf("the zero rejection lists %v, want nothing", got)
	}
	if (character.HigherLevelStartSubclassRequired{}).Classes() != nil {
		t.Error("the zero rejection owes no class")
	}
}
