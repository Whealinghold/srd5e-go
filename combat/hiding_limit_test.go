package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/hiding-searching-and-senses.md Limits (plan P10) - a hide scene holds at most 4 cover degrees and at most 256 enemy sightings.
// Path: unhappy (boundary).
func TestNewHideSceneCaps(t *testing.T) {
	t.Parallel()
	t.Run("cover", func(t *testing.T) {
		t.Parallel()
		runCap(t, "hide cover", 4, func(n int) error {
			cover := make(cvr, n)
			for i := range cover {
				cover[i] = combat.HideCoverDegreeHalfCover
			}
			_, err := combat.NewHideScene(gid("h"), content.ObscurementNone, cover, false, nil)
			return err
		})
	})
	t.Run("enemies", func(t *testing.T) {
		t.Parallel()
		runCap(t, "hide enemies", 256, func(n int) error {
			enemies := make([]combat.EnemySighting, n)
			for i := range enemies {
				enemies[i] = enemy(fmt.Sprintf("e%d", i), combat.LineOfSightBlocked, combat.SightRelationCannotSee)
			}
			_, err := combat.NewHideScene(gid("h"), content.ObscurementNone, nil, false, enemies)
			return err
		})
	})
}

// Requirement: doc/specs/hiding-searching-and-senses.md Limits (plan P10) - a sight scene holds at most 256 observer conditions and at most 256 target conditions.
// Path: unhappy (boundary).
func TestNewSightSceneCaps(t *testing.T) {
	t.Parallel()
	conds := func(n int) []content.ConditionKind {
		out := make([]content.ConditionKind, n)
		for i := range out {
			out[i] = content.ConditionPoisoned
		}
		return out
	}
	build := func(obs, tgt []content.ConditionKind) error {
		_, err := combat.NewSightScene(5, combat.LineOfSightClear, space(content.LightBright, 0, 0), senses(), obs, tgt)
		return err
	}
	t.Run("observer conditions", func(t *testing.T) {
		t.Parallel()
		runCap(t, "observer conditions", 256, func(n int) error { return build(conds(n), nil) })
	})
	t.Run("target conditions", func(t *testing.T) {
		t.Parallel()
		runCap(t, "target conditions", 256, func(n int) error { return build(nil, conds(n)) })
	})
}
