package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/derived-stats.md FR-040 - Armor Training is a set of
// categories that unions without altering either side (SRD 5.2.1, "Armor
// Training", p. 177).
// Path: unhappy (rule interaction).
func TestArmorTraining(t *testing.T) {
	t.Parallel()
	a := must(character.NewArmorTraining(content.ArmorHeavy, content.ArmorLight, content.ArmorHeavy))
	b := must(character.NewArmorTraining(content.ArmorShield))
	u := a.Union(b)
	if !a.Has(content.ArmorHeavy) || a.Has(content.ArmorShield) || !u.Has(content.ArmorShield) || u.Has(content.ArmorMedium) {
		t.Errorf("Has/Union wrong: %v %v", a.Categories(), u.Categories())
	}
	if want := []content.ArmorCategory{content.ArmorLight, content.ArmorHeavy, content.ArmorShield}; !slices.Equal(u.Categories(), want) {
		t.Errorf("Categories = %v, want %v", u.Categories(), want)
	}
	if got := (character.ArmorTraining{}).Categories(); len(got) != 0 {
		t.Errorf("empty set = %v", got)
	}
	if a.Has(0) {
		t.Error("invalid category reported as held")
	}
	_, err := character.NewArmorTraining(content.ArmorLight, 0)
	wantInvalid(t, err, "invalid category")
}

// Requirement: doc/specs/derived-stats.md FR-040 - a class's armorTraining
// converts by member name: a class trained in one category covers that
// category and no other (SRD 5.2.1, "Armor Training", p. 177).
// Path: per case.
func TestClassArmorTrainingConvertsByName(t *testing.T) {
	t.Parallel()
	armors := map[content.ArmorCategory]func(*testing.T) content.Armor{
		content.ArmorLight: leatherArmor, content.ArmorMedium: hideArmor, content.ArmorHeavy: chainMail,
	}
	wire := map[content.ArmorCategory]string{
		content.ArmorLight: "Light", content.ArmorMedium: "Medium", content.ArmorHeavy: "Heavy", content.ArmorShield: "Shield",
	}
	for trained := range wire {
		t.Run(wire[trained], func(t *testing.T) {
			t.Parallel()
			class := classFixture(t, "trainee", "D8", []string{"strength"}, []string{wire[trained]})
			for worn := range wire {
				in := base(t)
				in.ClassLevels = levels(cl("trainee", 1))
				in.InitialClass = class
				if worn == content.ArmorShield {
					in.Shield = shieldOf(t)
				} else {
					in.Armor = armors[worn](t)
				}
				var want []content.ArmorCategory
				if worn != trained {
					want = []content.ArmorCategory{worn}
				}
				if got := calculate(t, in).ArmorTrainingGaps(); !slices.Equal(got, want) {
					t.Errorf("trained in %s, wearing %v: gaps %v, want %v", wire[trained], worn, got, want)
				}
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-040 - the set operations leave their
// operands alone and order categories Light, Medium, Heavy, Shield.
// Path: unhappy (immutability).
func TestArmorTrainingSetOperations(t *testing.T) {
	t.Parallel()
	all := must(character.NewArmorTraining(content.ArmorShield, content.ArmorHeavy, content.ArmorMedium, content.ArmorLight))
	want := []content.ArmorCategory{content.ArmorLight, content.ArmorMedium, content.ArmorHeavy, content.ArmorShield}
	if !slices.Equal(all.Categories(), want) {
		t.Errorf("Categories = %v, want %v", all.Categories(), want)
	}
	for _, c := range want {
		only := must(character.NewArmorTraining(c))
		if got := only.Categories(); len(got) != 1 || got[0] != c || all.Union(only) != all || only.Union(only) != only {
			t.Errorf("single %v: %v", c, got)
		}
		if (character.ArmorTraining{}).Union(only) != only || only.Union(character.ArmorTraining{}) != only {
			t.Errorf("union with the empty set changed %v", c)
		}
	}
}
