package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func smSpeeds(tb testing.TB, walk int, fly int) combat.MovementSpeeds {
	tb.Helper()
	if fly < 0 {
		return luMust(combat.NewMovementSpeeds(walk))
	}
	return luMust(combat.NewMovementSpeeds(walk, combat.MovementSpeedEntry{Name: combat.SpeedNameFly, Feet: fly}))
}

// Requirement: doc/specs/spell-granted-modifiers.md table S rows S1 to S7, FR-050 to FR-055 - Longstrider adds 10 feet to every speed, Haste doubles every speed, Slow halves every speed rounded down; additions come before doubling and halving, Haste and Slow both apply, a Speed of 0 stays 0 and a speed the creature lacks stays absent (SRD 5.2.1, "Longstrider", p. 145; "Haste", p. 139; "Slow", p. 163; "Changes to Your Speeds", p. 188).
// Path: per case.
func TestQuerySpeeds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, row, path string
		spells          []content.SpellID
		walk, fly       int
		wantWalk        int
		wantFly         int
	}{
		{"Longstrider", "S1", "happy", []content.SpellID{"longstrider"}, 30, -1, 40, -1},
		{"Longstrider adds to a special speed", "S2", "unhappy (rule interaction)", []content.SpellID{"longstrider"}, 30, 60, 40, 70},
		{"Haste", "S3", "happy", []content.SpellID{"haste"}, 30, 60, 60, 120},
		{"Slow rounds down", "S4", "unhappy (boundary)", []content.SpellID{"slow"}, 35, -1, 17, -1},
		{"Longstrider then Haste", "S5", "unhappy (rule interaction)", []content.SpellID{"longstrider", "haste"}, 30, -1, 80, -1},
		{"Haste and Slow on 35", "S6", "unhappy (rule interaction)", []content.SpellID{"haste", "slow"}, 35, -1, 35, -1},
		{"Slow on a Speed of 0", "S7", "unhappy (empty and minimal)", []content.SpellID{"slow"}, 0, -1, 0, -1},
		{"Haste on a Speed of 0", "FR-055", "unhappy (boundary)", []content.SpellID{"haste"}, 0, -1, 0, -1},
		{"Slow halves a fly speed of 1 to 0", "FR-052", "unhappy (boundary)", []content.SpellID{"slow"}, 5, 1, 2, 0},
		{"all three: (30+10) x 2 / 2", "FR-054", "unhappy (rule interaction)", []content.SpellID{"slow", "haste", "longstrider"}, 30, -1, 40, -1},
		{"spells that change no speed", "FR-002", "unhappy (missing content)", []content.SpellID{"shield", "fireball"}, 30, 60, 30, 60},
		{"no spell", "FR-007", "unhappy (empty and minimal)", nil, 30, 60, 30, 60},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var src []combat.SpellModifierSource
			for i, s := range c.spells {
				src = append(src, smPlain(t, s, i))
			}
			got, err := combat.QuerySpeeds(src, cid("a"), smSpeeds(t, c.walk, c.fly))
			want := smSpeeds(t, c.wantWalk, c.wantFly)
			if err != nil || got != want {
				t.Errorf("speeds = %+v, %v, want %+v [%s%s]", got, err, want, spellModifiersSpec, c.row)
			}
			if f, has := got.Of(combat.SpeedNameFly); has != (c.fly >= 0) || (has && f != c.wantFly) {
				t.Errorf("fly = %d, %v", f, has)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-056, FR-010, FR-084 - two Longstrider on one creature are rejected as a duplicate (the host passes one, so +10 once); a source on another creature changes nothing; a zero creature is invalid input.
// Path: unhappy (rule interaction; invalid input).
func TestQuerySpeedsPerCreature(t *testing.T) {
	t.Parallel()
	base := smSpeeds(t, 30, -1)
	if got, err := combat.QuerySpeeds([]combat.SpellModifierSource{smSrc(t, "haste", "c", "b", 0, combat.NoChoice{})}, cid("a"), base); err != nil || got != base {
		t.Errorf("Haste on b changed a's speeds: %+v, %v", got, err)
	}
	if got, err := combat.QuerySpeeds([]combat.SpellModifierSource{smPlain(t, "longstrider", 0)}, cid("a"), base); err != nil || got.Walk() != 40 {
		t.Errorf("one Longstrider = %+v, %v, want 40", got, err)
	}
	if _, err := combat.QuerySpeeds([]combat.SpellModifierSource{smPlain(t, "longstrider", 0), smSrc(t, "longstrider", "d", "a", 1, combat.NoChoice{})}, cid("a"), base); !errors.Is(err, combat.ErrDuplicateSource) {
		t.Errorf("two Longstrider = %v, want ErrDuplicateSource", err)
	}
	if _, err := combat.QuerySpeeds(nil, combat.CombatantID(""), base); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero creature = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-053, FR-055, plan-phase-4.md "Limits" - the arithmetic saturates at the integer limits instead of wrapping: Longstrider and Haste on a Speed near math.MaxInt stay positive; Slow never goes below 0.
// Path: unhappy (boundary).
func TestQuerySpeedsSaturate(t *testing.T) {
	t.Parallel()
	huge := luMust(combat.NewMovementSpeeds(math.MaxInt))
	got, err := combat.QuerySpeeds([]combat.SpellModifierSource{smPlain(t, "longstrider", 0), smPlain(t, "haste", 1)}, cid("a"), huge)
	if err != nil || got.Walk() != math.MaxInt {
		t.Errorf("Longstrider and Haste on the maximum = %d, %v, want the maximum", got.Walk(), err)
	}
	if got, _ := combat.QuerySpeeds([]combat.SpellModifierSource{smPlain(t, "slow", 0)}, cid("a"), huge); got.Walk() != math.MaxInt/2 {
		t.Errorf("Slow on the maximum = %d, want half", got.Walk())
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-053, conditions.md D-8 - rule interactions between active spells and conditions: the spell speeds go through the conditions after, so a Grappled creature stays at 0 under Haste and Exhaustion takes 5 feet per level from the doubled Speed, never below 0 (SRD 5.2.1, "Haste", p. 139; "Grappled", p. 185).
// Path: unhappy (rule interaction).
func TestSpellSpeedsThenConditions(t *testing.T) {
	t.Parallel()
	haste := []combat.SpellModifierSource{smPlain(t, "haste", 0)}
	hasted := luMust(combat.QuerySpeeds(haste, cid("a"), smSpeeds(t, 30, -1)))
	grappled := condSet(t, condConds(combat.Grappled{Grappler: idB()}))
	if sp, err := (combat.ConditionEffects{}).Speed(grappled, hasted.Walk()); err != nil || sp.Value != 0 {
		t.Errorf("hasted and Grappled = %+v, %v, want 0", sp, err)
	}
	tired := condSet(t, condConds(condLevel(2)))
	if sp, err := (combat.ConditionEffects{}).Speed(tired, hasted.Walk()); err != nil || sp.Value != 50 {
		t.Errorf("hasted and Exhaustion 2 = %+v, %v, want 60 - 10", sp, err)
	}
}
