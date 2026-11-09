package character_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/character-aggregate.md FR-020, FR-021, FR-023, FR-097 -
// an award adds to the total, emits one ExperiencePointsGained, changes nothing
// else and may pass 355,000 without a level; an overflowing total saturates.
// Path: happy.
func TestCharacterAwardExperience(t *testing.T) {
	t.Parallel()
	ch := restored(t, 300, 12, flatScores(t, 10), cl("fighter", 1))
	before := ch
	got := ch.AwardExperience(character.MustExperienceAmount(200))
	n := got.Character()
	evs := got.Events()
	ev, ok := evs[0].(character.ExperiencePointsGained)
	if n.ExperiencePoints().Value() != 500 || len(evs) != 1 || !ok || ev.Amount().Value() != 200 || ev.Total().Value() != 500 || ev.CharacterID() != ch.ID() {
		t.Errorf("award: %v %#v", n.ExperiencePoints().Value(), evs)
	}
	if n.TotalLevel() != ch.TotalLevel() || !n.ClassLevels().Equal(ch.ClassLevels()) || n.HitPointMaximum() != ch.HitPointMaximum() || !n.HitDicePool().Equal(ch.HitDicePool()) ||
		n.HeroicInspiration() != ch.HeroicInspiration() || !n.FeatureStates().Equal(ch.FeatureStates()) {
		t.Error("FR-021: nothing but the XP may change")
	}
	if !ch.Equal(before) {
		t.Error("the receiver must not change")
	}
	big := restored(t, 354000, 300, flatScores(t, 10), cl("fighter", 20)).AwardExperience(character.MustExperienceAmount(1000000))
	if big.Character().ExperiencePoints().Value() != 1354000 || big.Character().TotalLevel().Value() != 20 {
		t.Error("FR-023: awards past 355,000 succeed")
	}
	sat := restored(t, 5, 300, flatScores(t, 10), cl("fighter", 20)).AwardExperience(character.MustExperienceAmount(math.MaxInt))
	if sat.Character().ExperiencePoints().Value() != math.MaxInt {
		t.Error("an overflowing award must saturate")
	}
}
