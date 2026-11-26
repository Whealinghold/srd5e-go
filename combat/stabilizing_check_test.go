package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-069 - StabilizingCheck is DC 10, Wisdom, Medicine (SRD
// 5.2.1, "Stabilizing a Character", p. 18); the Medicine skill uses Wisdom.
// Path: happy.
func TestStabilizingCheck(t *testing.T) {
	t.Parallel()
	c := combat.StabilizingCheck{}
	if c.DC() != 10 || c.Ability() != ability.Wisdom || c.Skill() != content.SkillMedicine || c.Skill().Ability() != c.Ability() {
		t.Errorf("check = DC %d %v %v", c.DC(), c.Ability(), c.Skill())
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-072, table 4.8a - a StableRecoveryDelay is 1 to 4 hours.
// Path: unhappy (boundary).
func TestStableRecoveryDelay(t *testing.T) {
	t.Parallel()
	for h := 1; h <= 4; h++ {
		if d, err := combat.NewStableRecoveryDelay(h); err != nil || d.Hours() != h {
			t.Errorf("hours %d = %+v, %v", h, d, err)
		}
	}
	for _, h := range []int{0, 5, -1} {
		if _, err := combat.NewStableRecoveryDelay(h); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("hours %d error = %v, want ErrInvalidInput", h, err)
		}
	}
}
