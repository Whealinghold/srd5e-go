package character_test

import (
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: doc/specs/magic-item-effects.md FR-022, FR-023, FR-025 (SRD 5.2.1, "Potions of Healing", p. 236; "Potion of Healing", p. 99) - table
// "Potion of Healing grades": the four grades have their dice, rarity and range, ForRarity maps all six rarities (Legendary and Artifact are not
// listed), and the standard grade equals the gear row 2d4 + 2.
// Path: per case.
func TestPotionOfHealingGrades(t *testing.T) {
	t.Parallel()
	rows := []struct {
		grade  character.PotionOfHealingGrade
		dice   dice.Expression
		rarity content.ItemRarity
		lo, hi int
		name   string
	}{
		{character.GradeStandard, dice.MustExpression(2, dice.D4(), 2), content.RarityCommon, 4, 10, "standard"},
		{character.GradeGreater, dice.MustExpression(4, dice.D4(), 4), content.RarityUncommon, 8, 20, "greater"},
		{character.GradeSuperior, dice.MustExpression(8, dice.D4(), 8), content.RarityRare, 16, 40, "superior"},
		{character.GradeSupreme, dice.MustExpression(10, dice.D4(), 20), content.RarityVeryRare, 30, 60, "supreme"},
	}
	for _, r := range rows {
		if r.grade.Dice() != r.dice || r.grade.Rarity() != r.rarity || r.grade.Dice().Minimum() != r.lo || r.grade.Dice().Maximum() != r.hi || r.grade.String() != r.name {
			t.Errorf("%s: %v %v %d-%d %s", r.name, r.grade.Dice(), r.grade.Rarity(), r.grade.Dice().Minimum(), r.grade.Dice().Maximum(), r.grade)
		}
		found, ok := (character.PotionOfHealingGrades{}).ForRarity(r.rarity).(character.PotionOfHealingGradeFound)
		if !ok || found.Grade() != r.grade {
			t.Errorf("ForRarity(%s) = %+v", r.rarity, found)
		}
	}
	for _, r := range []content.ItemRarity{content.RarityLegendary, content.RarityArtifact, 0, content.RarityArtifact + 1} {
		nl, ok := (character.PotionOfHealingGrades{}).ForRarity(r).(character.PotionOfHealingGradeNotListed)
		if !ok || nl.Rarity() != r {
			t.Errorf("ForRarity(%v) = %+v", r, nl)
		}
	}
	for _, g := range []character.PotionOfHealingGrade{0, character.GradeSupreme + 1} {
		if !g.Dice().IsZero() || g.Rarity() != 0 || g.String() != "unknown" {
			t.Errorf("invalid grade %d", g)
		}
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-026 (SRD 5.2.1, "Potion of Resistance", p. 236) - the 1d10 table maps 1 to 10 to the ten types in
// order and 0, 11 and a negative roll return a zero value and an error matching ErrInvalidInput.
// Path: unhappy (boundary).
func TestResistanceTypeTable(t *testing.T) {
	t.Parallel()
	want := []content.DamageType{content.DamageAcid, content.DamageCold, content.DamageFire, content.DamageForce, content.DamageLightning,
		content.DamageNecrotic, content.DamagePoison, content.DamagePsychic, content.DamageRadiant, content.DamageThunder}
	for i, w := range want {
		if got, err := (character.ResistanceTypeTable{}).ForRoll(i + 1); err != nil || got != w {
			t.Errorf("roll %d = %v %v, want %v", i+1, got, err, w)
		}
	}
	for _, roll := range []int{0, 11, -1} {
		got, err := (character.ResistanceTypeTable{}).ForRoll(roll)
		if got != 0 {
			t.Errorf("roll %d = %v", roll, got)
		}
		wantInvalid(t, err, "roll")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-027 (SRD 5.2.1, "Potion of Resistance", p. 236) - a chosen type must be one of the ten; Bludgeoning,
// Piercing and Slashing are refused, a negative duration is refused, the potion lasts 60 minutes and a ring or armor lasts while worn (zero).
// Path: unhappy (invalid input).
func TestGrantedResistancesChosen(t *testing.T) {
	t.Parallel()
	rules := character.GrantedResistances{}
	for _, d := range []content.DamageType{content.DamageAcid, content.DamageThunder, content.DamageFire} {
		g, ok := rules.Chosen(d, character.ResistancePotionDuration).(character.GrantedResistance)
		if !ok || g.DamageType() != d || g.Duration() != time.Hour {
			t.Errorf("%v = %+v", d, g)
		}
	}
	if g, ok := rules.Chosen(content.DamageCold, 0).(character.GrantedResistance); !ok || g.Duration() != 0 {
		t.Errorf("while wearing = %+v", g)
	}
	for _, d := range []content.DamageType{content.DamageBludgeoning, content.DamagePiercing, content.DamageSlashing, 0, content.DamageThunder + 1} {
		r, ok := rules.Chosen(d, time.Hour).(character.GrantedResistanceRejected)
		if !ok || r.Reason() != character.ResistanceTypeNotListed {
			t.Errorf("%v = %+v", d, r)
		}
	}
	r, ok := rules.Chosen(content.DamageFire, -time.Second).(character.GrantedResistanceRejected)
	if !ok || r.Reason() != character.ResistanceDurationNegative {
		t.Errorf("negative duration = %+v", r)
	}
	if character.ResistanceTypeNotListed.String() != "type_not_listed" || character.ResistanceDurationNegative.String() != "duration_negative" || character.GrantedResistanceRejection(0).String() != "unknown" {
		t.Error("rejection names")
	}
}
