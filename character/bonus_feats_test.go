package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case
// Path: per case.
func TestBonusFeatsRemaining(t *testing.T) {
	t.Parallel()
	bonus := character.BonusFeat{}
	asiSrc := character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_asi"}
	take := func(n int) character.TakenFeats {
		// distinct ids keep NewTakenFeats valid; only the source matters to the count
		feats := []character.TakenFeat{must(character.NewTakenFeat("defense", content.FeatFightingStyle, asiSrc, character.NoFeatChoices{}))}
		for _, id := range []content.FeatID{"alert", "savage_attacker"}[:n] {
			feats = append(feats, must(character.NewTakenFeat(id, content.FeatOrigin, bonus, character.NoFeatChoices{})))
		}
		return must(character.NewTakenFeats(feats...))
	}
	xp := func(n int) character.ExperiencePoints { return must(character.NewExperiencePoints(n)) }
	for _, c := range []struct {
		name, requirement, path string
		option                  character.BonusFeatsAtLevel20
		xp                      int
		taken                   int
		want                    int
	}{
		{"NoBonusFeats gives 0", featsSpec + ` FR-084; SRD 5.2.1, "Bonus Feats at Level 20", p. 24`, "happy", character.NoBonusFeats{}, 500000, 0, 0},
		{"a nil option gives 0", featsSpec + " FR-084", "unhappy (invalid input)", nil, 500000, 0, 0},
		{"355,000", featsSpec + " FR-084", "unhappy (boundary)", character.BonusFeatsFromXp{}, 355000, 0, 0},
		{"384,999", featsSpec + " FR-084", "unhappy (boundary)", character.BonusFeatsFromXp{}, 384999, 0, 0},
		{"385,000", featsSpec + " FR-084", "unhappy (boundary)", character.BonusFeatsFromXp{}, 385000, 0, 1},
		{"415,000", featsSpec + " FR-084", "unhappy (boundary)", character.BonusFeatsFromXp{}, 415000, 0, 2},
		{"385,000 with one taken", featsSpec + " FR-084", "unhappy (boundary)", character.BonusFeatsFromXp{}, 385000, 1, 0},
		{"415,000 with one taken", featsSpec + " FR-084", "unhappy (boundary)", character.BonusFeatsFromXp{}, 415000, 1, 1},
		{"415,000 with two taken", featsSpec + " FR-084", "unhappy (boundary)", character.BonusFeatsFromXp{}, 415000, 2, 0},
		{"never negative", featsSpec + " FR-084", "unhappy (boundary)", character.BonusFeatsFromXp{}, 355000, 2, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (character.BonusFeats{}).Remaining(c.option, xp(c.xp), take(c.taken)); got != c.want {
				t.Errorf("Remaining = %d, want %d [%s]", got, c.want, c.requirement)
			}
		})
	}
}
