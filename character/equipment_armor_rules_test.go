package character_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: per case equipSpec FR-001, FR-002, FR-003, FR-004, FR-006
// (SRD 5.2.1, Armor table, p. 92) - the "Armor base" rows: listed base plus
// the contribution gives the total.
// Path: per case (happy and unhappy boundary and rule interaction).
func TestBaseArmorClass(t *testing.T) {
	t.Parallel()
	type tc struct {
		name         string
		rule         content.BaseRule
		dex          int
		listed, dexC int
	}
	full := func(b int) content.BaseRule { return fullRule(t, b) }
	capd := func(b, m int) content.BaseRule { return cappedRule(t, b, m) }
	none := func(b int) content.BaseRule { return noneRule(t, b) }
	cases := []tc{
		{"Leather Armor +3", full(11), 3, 11, 3}, {"Leather Armor 0", full(11), 0, 11, 0},
		{"Padded Armor -1", full(11), -1, 11, -1}, {"Padded Armor -5", full(11), -5, 11, -5},
		{"Studded Leather +5", full(12), 5, 12, 5}, {"Studded Leather +10", full(12), 10, 12, 10},
		{"Hide Armor +1", capd(12, 2), 1, 12, 1}, {"Hide Armor +2", capd(12, 2), 2, 12, 2},
		{"Hide Armor +3", capd(12, 2), 3, 12, 2}, {"Half Plate +5", capd(15, 2), 5, 15, 2},
		{"Chain Shirt 0", capd(13, 2), 0, 13, 0}, {"Breastplate -2", capd(14, 2), -2, 14, -2},
		{"Scale Mail -5", capd(14, 2), -5, 14, -5},
		{"Ring Mail +5", none(14), 5, 14, 0}, {"Chain Mail -3", none(16), -3, 16, 0},
		{"Splint Armor +10", none(17), 10, 17, 0}, {"Plate Armor 0", none(18), 0, 18, 0},
		{"cap 0, +3", capd(10, 0), 3, 10, 0}, {"cap 0, -1", capd(10, 0), -1, 10, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.ArmorRules{}.BaseArmorClass(c.rule, modifierOf(t, c.dex))
			if got.ListedBase() != c.listed || got.DexterityContribution() != c.dexC || got.Total() != c.listed+c.dexC {
				t.Errorf("base = %d + %d = %d, want %d + %d", got.ListedBase(), got.DexterityContribution(), got.Total(), c.listed, c.dexC)
			}
		})
	}
}

// Requirement: equipSpec FR-003 (SRD 5.2.1, Armor table, p. 92) - Heavy armor
// gives its listed base for every modifier from -5 to +10.
// Path: unhappy (rule interaction: Dexterity ignored).
func TestBaseArmorClassHeavyIgnoresDexterity(t *testing.T) {
	t.Parallel()
	rule := noneRule(t, 18)
	for m := -5; m <= 10; m++ {
		if got := (character.ArmorRules{}).BaseArmorClass(rule, modifierOf(t, m)).Total(); got != 18 {
			t.Errorf("Plate Armor with %+d = %d, want 18", m, got)
		}
	}
}

// Requirement: equipSpec FR-004 - ArmorBase is a comparable value: equal parts
// are equal under ==.
// Path: happy.
func TestArmorBaseEquality(t *testing.T) {
	t.Parallel()
	a := character.ArmorRules{}.BaseArmorClass(cappedRule(t, 12, 2), modifierOf(t, 3))
	b := character.ArmorRules{}.BaseArmorClass(cappedRule(t, 12, 2), modifierOf(t, 2))
	c := character.ArmorRules{}.BaseArmorClass(fullRule(t, 12), modifierOf(t, 3))
	if a != b || a == c {
		t.Errorf("capped +3 %v and capped +2 %v should be equal and differ from full +3 %v", a, b, c)
	}
	var zero content.BaseRule
	if got := (character.ArmorRules{}).BaseArmorClass(zero, modifierOf(t, 3)); got != (character.ArmorBase{}) {
		t.Errorf("zero rule = %v, want the zero base", got)
	}
}

// Requirement: equipSpec FR-005 (content-equipment.md FR-024) - the closed
// ArmorClassRule has exactly BaseRule and BonusRule and a BaseRule carries one
// of three treatments; ArmorRules evaluates only a BaseRule, so a Shield cannot
// be passed.
// Path: unhappy (rule interaction).
func TestArmorRulesEvaluateOnlyBaseRules(t *testing.T) {
	t.Parallel()
	for _, rule := range []content.ArmorClassRule{fullRule(t, 11), must(content.NewBonusRule(2))} {
		switch rule.(type) {
		case content.BaseRule:
		case content.BonusRule:
		}
	}
	for _, d := range []content.DexterityBonus{content.FullDexterity{}, content.CappedDexterity{}, content.NoDexterity{}} {
		switch d.(type) {
		case content.FullDexterity:
		case content.CappedDexterity:
		case content.NoDexterity:
		}
	}
	if _, ok := shieldOf(t).ArmorClass().(content.BaseRule); ok {
		t.Error("a Shield's rule is a BonusRule, not a BaseRule")
	}
}

// Requirement: per case equipSpec FR-007, FR-008, FR-009, FR-010 (SRD 5.2.1,
// "Armor", Strength, p. 92) - the Strength requirement rows.
// Path: per case (unhappy boundary; empty and minimal).
func TestSpeedPenaltyFeet(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		need     int
		strength int
		want     int
	}{
		{"Chain Mail (13), 12", 13, 12, 10}, {"Chain Mail (13), 13", 13, 13, 0}, {"Chain Mail (13), 14", 13, 14, 0},
		{"Chain Mail (13), 1", 13, 1, 10}, {"Chain Mail (13), 30", 13, 30, 0},
		{"Splint Armor (15), 14", 15, 14, 10}, {"Splint Armor (15), 15", 15, 15, 0},
		{"Plate Armor (15), 8", 15, 8, 10}, {"Plate Armor (15), 15", 15, 15, 0},
		{"Padded Armor (none), 30", 0, 30, 0}, {"Ring Mail (none), 1", 0, 1, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := armorOf(t, "armor_x", content.ArmorHeavy, noneRule(t, 16), c.need, false, 40, gp(75))
			if got := (character.ArmorRules{}).SpeedPenaltyFeet(a, score(t, c.strength)); got != c.want {
				t.Errorf("penalty = %d, want %d", got, c.want)
			}
		})
	}
}

// Requirement: equipSpec FR-009, FR-010 - the penalty is 10 for scores below
// the requirement and 0 from it to 30, for requirements 13 and 15 and for none.
// Path: unhappy (boundary; empty and minimal).
func TestSpeedPenaltySweep(t *testing.T) {
	t.Parallel()
	for _, need := range []int{0, 13, 15} {
		a := armorOf(t, "armor_x", content.ArmorHeavy, noneRule(t, 16), need, false, 40, gp(75))
		for s := ability.MinScore; s <= ability.MaxScore; s++ {
			want := 0
			if need != 0 && s < need {
				want = 10
			}
			if got := (character.ArmorRules{}).SpeedPenaltyFeet(a, score(t, s)); got != want {
				t.Errorf("requirement %d, score %d: %d, want %d", need, s, got, want)
			}
		}
	}
}

// Requirement: per case equipSpec FR-079 (SRD 5.2.1, "Armor", Stealth, p. 92;
// "They Don't Stack", p. 8) - one Disadvantage when any worn armor is flagged,
// else none; a fresh slice.
// Path: unhappy (rule interaction).
func TestStealthCircumstances(t *testing.T) {
	t.Parallel()
	padded := armorOf(t, "padded_x", content.ArmorLight, fullRule(t, 11), 0, true, 8, gp(5))
	plate := armorOf(t, "plate_x", content.ArmorHeavy, noneRule(t, 18), 15, true, 65, gp(1500))
	leather := armorOf(t, "leather_x", content.ArmorLight, fullRule(t, 11), 0, false, 10, gp(10))
	chain := armorOf(t, "chain_x", content.ArmorHeavy, noneRule(t, 16), 13, true, 55, gp(75))
	scale := armorOf(t, "scale_x", content.ArmorMedium, cappedRule(t, 14, 2), 0, true, 45, gp(50))
	shield := shieldOf(t)
	cases := []struct {
		name string
		worn []content.Armor
		want int
	}{
		{"Padded Armor", []content.Armor{padded}, 1}, {"Plate Armor", []content.Armor{plate}, 1},
		{"Leather Armor", []content.Armor{leather}, 0}, {"Chain Mail + Shield", []content.Armor{chain, shield}, 1},
		{"Scale Mail + Plate Armor", []content.Armor{scale, plate}, 1}, {"Shield alone", []content.Armor{shield}, 0},
		{"no armor", nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.ArmorRules{}.StealthCircumstances(c.worn)
			if len(got) != c.want || (c.want == 1 && got[0] != (modifier.Disadvantage{})) {
				t.Errorf("circumstances = %v, want %d Disadvantage", got, c.want)
			}
			if c.want == 1 {
				got[0] = modifier.Advantage{} // a fresh slice: the next read is unchanged
				if again := (character.ArmorRules{}).StealthCircumstances(c.worn); again[0] != (modifier.Disadvantage{}) {
					t.Errorf("mutating the result changed the next read: %v", again)
				}
			}
		})
	}
	if got := fmt.Sprint(character.ArmorRules{}.StealthCircumstances(nil)); got != "[]" {
		t.Errorf("no worn armor = %s", got)
	}
}
