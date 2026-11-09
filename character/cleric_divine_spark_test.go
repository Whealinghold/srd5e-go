package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-cleric.md FR-049, FR-050, FR-051, FR-052, FR-053, table 4.7.1 DS-01 to DS-12 (SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37) - the
// d8s rolled at each level band plus the Wisdom modifier give the total; healing reports the total (not below 0), damage reports the total on a failed save and half of it,
// rounded down, on a successful one; a negative total gives 0 in both modes.
// Path: per case (DS-01 to DS-12): happy / unhappy (boundary).
func TestDivineSparkResolve(t *testing.T) {
	t.Parallel()
	heal, dmg := character.SparkHealing, character.SparkDamage
	for _, c := range []struct {
		label                      string
		level                      int
		rolls                      []int
		wis                        int
		mode                       character.SparkMode
		total, healing, fail, pass int
	}{
		{"DS-01", 2, []int{5}, 3, heal, 8, 8, 0, 0}, {"DS-02", 2, []int{5}, 3, dmg, 8, 0, 8, 4}, {"DS-03", 6, []int{4}, 2, dmg, 6, 0, 6, 3},
		{"DS-04", 7, []int{3, 8}, 4, dmg, 15, 0, 15, 7}, {"DS-05", 12, []int{1, 1}, 0, heal, 2, 2, 0, 0}, {"DS-06", 13, []int{8, 8, 8}, 5, dmg, 29, 0, 29, 14},
		{"DS-07", 17, []int{2, 2, 2}, 1, heal, 7, 7, 0, 0}, {"DS-08", 18, []int{1, 1, 1, 1}, 5, dmg, 9, 0, 9, 4}, {"DS-09", 20, []int{8, 8, 8, 8}, 5, dmg, 37, 0, 37, 18},
		{"DS-10", 2, []int{1}, 0, dmg, 1, 0, 1, 0}, {"DS-11", 2, []int{1}, -1, heal, 0, 0, 0, 0}, {"DS-12", 2, []int{2}, -5, dmg, -3, 0, 0, 0},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			r, err := character.DivineSpark{}.Resolve(character.MustClericLevel(c.level), c.rolls, modifierOf(t, c.wis), c.mode, content.DamageRadiant)
			if err != nil {
				t.Fatal(err)
			}
			if r.Total() != c.total || r.HitPointsToRestore() != c.healing || r.DamageOnFailedSave() != c.fail || r.DamageOnSuccessfulSave() != c.pass || r.Mode() != c.mode || !slices.Equal(r.Rolls(), c.rolls) {
				t.Errorf("= total %d, heal %d, fail %d, pass %d", r.Total(), r.HitPointsToRestore(), r.DamageOnFailedSave(), r.DamageOnSuccessfulSave())
			}
			if (c.mode == dmg) != (r.DamageType() == content.DamageRadiant) {
				t.Errorf("damage type = %v in mode %v", r.DamageType(), c.mode)
			}
			r.Rolls()[0] = 99
			if r.Rolls()[0] == 99 {
				t.Error("Rolls returned a shared slice")
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-049, FR-052, FR-054, table 4.7.1 DS-13 and DS-14 - a roll outside 1 to 8, a number of rolls other than the level's dice, the zero level, an
// invalid mode and a damage type other than Necrotic or Radiant (damage mode) are errors matching srderr.ErrInvalidInput.
// Path: per case (unhappy: invalid input).
func TestDivineSparkResolveRefusals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		level character.ClericLevel
		rolls []int
		mode  character.SparkMode
		dt    content.DamageType
	}{
		{"DS-13 a 9", character.MustClericLevel(2), []int{9}, character.SparkHealing, 0},
		{"a 0", character.MustClericLevel(2), []int{0}, character.SparkHealing, 0},
		{"DS-14 one value for two dice", character.MustClericLevel(7), []int{4}, character.SparkDamage, content.DamageRadiant},
		{"too many values", character.MustClericLevel(6), []int{4, 4}, character.SparkHealing, 0},
		{"zero level", character.ClericLevel{}, nil, character.SparkHealing, 0},
		{"no mode", character.MustClericLevel(2), []int{4}, 0, 0},
		{"Fire damage", character.MustClericLevel(2), []int{4}, character.SparkDamage, content.DamageFire},
		{"no damage type", character.MustClericLevel(2), []int{4}, character.SparkDamage, 0},
	} {
		_, err := character.DivineSpark{}.Resolve(c.level, c.rolls, modifierOf(t, 0), c.mode, c.dt)
		wantInvalid(t, err, c.name)
	}
	// Healing ignores the damage type.
	if _, err := (character.DivineSpark{}).Resolve(character.MustClericLevel(2), []int{4}, modifierOf(t, 0), character.SparkHealing, content.DamageFire); err != nil {
		t.Errorf("healing with a stray damage type: %v", err)
	}
}

// Requirement: doc/specs/class-cleric.md FR-049, FR-052 - the dice count is 1, 2, 3, 4 at the boundary levels, 0 for the zero level; only Necrotic and Radiant are allowed.
// Path: per case (unhappy: boundary).
func TestDivineSparkDiceAndDamageTypes(t *testing.T) {
	t.Parallel()
	for level, want := range map[int]int{2: 1, 6: 1, 7: 2, 12: 2, 13: 3, 17: 3, 18: 4, 20: 4} {
		if got := (character.DivineSpark{}).Dice(character.MustClericLevel(level)); got != want {
			t.Errorf("Dice(%d) = %d, want %d", level, got, want)
		}
	}
	if (character.DivineSpark{}).Dice(character.ClericLevel{}) != 0 {
		t.Error("Dice of the zero level")
	}
	for dt := content.DamageAcid; dt <= content.DamageThunder; dt++ {
		want := dt == content.DamageNecrotic || dt == content.DamageRadiant
		if (character.DivineSpark{}).AllowsDamageType(dt) != want {
			t.Errorf("AllowsDamageType(%v) = %v", dt, !want)
		}
	}
	if (character.DivineSpark{}).AllowsDamageType(0) {
		t.Error("the zero damage type is allowed")
	}
}

// Requirement: doc/specs/class-cleric.md FR-056, FR-057, FR-128, FR-132, table 4.7.2 DT-01 to DT-07 (SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37) - the target
// is another visible creature within 30 feet; the first reason is target_is_self, target_not_visible, then target_out_of_range; a negative distance is invalid input.
// Path: per case (DT-01 to DT-07): unhappy (boundary, invalid input, rule interaction).
func TestDivineSparkTargetProblem(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label  string
		target character.DivineSparkTarget
		want   character.ClericReason
	}{
		{"DT-01", character.DivineSparkTarget{Visible: true, DistanceFeet: 30}, 0},
		{"DT-02", character.DivineSparkTarget{Visible: true, DistanceFeet: 31}, character.ClericTargetOutOfRange},
		{"DT-03", character.DivineSparkTarget{Visible: true, DistanceFeet: 0}, 0},
		{"DT-04", character.DivineSparkTarget{IsSelf: true, Visible: true}, character.ClericTargetIsSelf},
		{"DT-05", character.DivineSparkTarget{DistanceFeet: 10}, character.ClericTargetNotVisible},
		{"DT-06", character.DivineSparkTarget{IsSelf: true, DistanceFeet: 40}, character.ClericTargetIsSelf},
		{"DT-07", character.DivineSparkTarget{DistanceFeet: 40}, character.ClericTargetNotVisible},
	} {
		got, err := character.DivineSpark{}.TargetProblem(c.target)
		if err != nil || got != c.want {
			t.Errorf("%s: = %v, %v; want %v", c.label, got, err, c.want)
		}
	}
	_, err := character.DivineSpark{}.TargetProblem(character.DivineSparkTarget{Visible: true, DistanceFeet: -1})
	wantInvalid(t, err, "negative distance")
}

// Requirement: doc/specs/class-cleric.md FR-114, FR-115, FR-116, table 4.12.4 SH-01 to SH-12 (SRD 5.2.1, "Level 17: Supreme Healing", p. 40) - Supreme Healing applies from Cleric
// level 17 to a Life Domain Cleric only (16, no subclass and another subclass roll); the highest number is count times sides (2d6 gives 12) and fewer than 1 die or fewer than 2
// sides is invalid input.
// Path: per case (SH rows): happy / unhappy (boundary, invalid input, state error).
func TestSupremeHealing(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label    string
		level    int
		subclass content.SubclassID
		want     bool
	}{
		{"SH-01", 17, "life_domain", true}, {"SH-03", 16, "life_domain", false}, {"SH-06", 20, "life_domain", true}, {"SH-09", 17, "", false}, {"SH-10", 17, "other", false},
	} {
		if got := (character.SupremeHealing{}).Applies(character.MustClericLevel(c.level), c.subclass); got != c.want {
			t.Errorf("%s: Applies = %v", c.label, got)
		}
	}
	for _, c := range []struct {
		label        string
		count, sides int
		want         int
		ok           bool
	}{{"SH-01", 2, 6, 12, true}, {"SH-02", 1, 4, 4, true}, {"SH-04", 3, 8, 24, true}, {"SH-07", 0, 6, 0, false}, {"SH-08", 1, 1, 0, false}, {"negative", -1, 6, 0, false}} {
		got, err := character.SupremeHealing{}.Maximum(c.count, c.sides)
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("%s: Maximum = %d, %v", c.label, got, err)
		}
		if err != nil {
			wantInvalid(t, err, c.label)
		}
	}
}
