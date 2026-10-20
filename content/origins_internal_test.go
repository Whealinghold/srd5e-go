package content

import "testing"

// The valid methods of the unexported-field values guard a zero or partly
// built value that a constructor never returns, so only an internal test can
// reach each half of their conditions.

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-069 - a value with one bad part is not valid,
// whichever part it is, so a zero or hand-built option is rejected by NewSpeciesChoice and by NewSpecies.
// Path: per case.
func TestChoiceOptionValidity(t *testing.T) {
	t.Parallel()
	good := LevelBenefit{level: 1, text: "t", spellID: "spell_a"}
	three := func(a, b, c LevelBenefit) [3]LevelBenefit { return [3]LevelBenefit{a, b, c} }
	l := func(n int) LevelBenefit { return LevelBenefit{level: n, text: "t", spellID: "spell_a"} }
	cases := []struct {
		name string
		got  bool
		want bool
	}{
		{"damage option", DamageTypeOption{name: "N", damageType: DamageAcid}.valid(), true},
		{"damage option, bad name", DamageTypeOption{name: " ", damageType: DamageAcid}.valid(), false},
		{"damage option, bad type", DamageTypeOption{name: "N"}.valid(), false},
		{"benefit", good.valid(), true},
		{"benefit, bad level", LevelBenefit{level: 2, text: "t", spellID: "spell_a"}.valid(), false},
		{"benefit, bad text", LevelBenefit{level: 1, text: " ", spellID: "spell_a"}.valid(), false},
		{"benefit, bad spell", LevelBenefit{level: 1, text: "t"}.valid(), false},
		{"lineage", LineageOption{name: "N", benefits: three(l(1), l(3), l(5))}.valid(), true},
		{"lineage, bad name", LineageOption{name: "", benefits: three(l(1), l(3), l(5))}.valid(), false},
		{"lineage, bad first benefit", LineageOption{name: "N", benefits: three(LevelBenefit{}, l(3), l(5))}.valid(), false},
		{"lineage, middle benefit with a valid level and no text", LineageOption{name: "N", benefits: three(l(1), LevelBenefit{level: 3, spellID: "spell_a"}, l(5))}.valid(), false},
		{"lineage, bad last benefit", LineageOption{name: "N", benefits: three(l(1), l(3), LevelBenefit{})}.valid(), false},
		{"lineage, levels out of order", LineageOption{name: "N", benefits: three(l(1), l(5), l(3))}.valid(), false},
		{"lineage, repeated level", LineageOption{name: "N", benefits: three(l(1), l(3), l(3))}.valid(), false},
		{"text option", TextOption{name: "N", text: "t", spellIDs: []SpellID{"spell_a"}}.valid(), true},
		{"text option, bad name", TextOption{name: "", text: "t"}.valid(), false},
		{"text option, bad text", TextOption{name: "N", text: " "}.valid(), false},
		{"text option, bad spell", TextOption{name: "N", text: "t", spellIDs: []SpellID{"spell_a", "Bad"}}.valid(), false},
		{"choice", SpeciesChoice{traitName: "T", options: []ChoiceOption{TextOption{name: "N", text: "t"}}}.valid(), true},
		{"choice, no options", SpeciesChoice{traitName: "T"}.valid(), false},
		{"feat benefit", FeatBenefit{text: "t"}.valid(), true},
		{"feat benefit, bad text", FeatBenefit{name: "N"}.valid(), false},
		{"feat benefit, bad name", FeatBenefit{name: " N", text: "t"}.valid(), false},
		{"equipment item", EquipmentItem{name: "N", quantity: 1}.valid(), true},
		{"equipment item, bad name", EquipmentItem{name: "", quantity: 1}.valid(), false},
		{"equipment item, zero quantity", EquipmentItem{name: "N"}.valid(), false},
		{"equipment item, quantity at the cap", EquipmentItem{name: "N", quantity: MaxEquipmentQuantity}.valid(), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Errorf("valid() = %v, want %v [FR-069, unhappy (invalid input)]", c.got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-014 - a trait list needs valid parts: a name
// without text and a text without a name are both rejected.
// Path: unhappy (invalid input).
func TestCheckNamedTextsRejectsPartlyBuiltTraits(t *testing.T) {
	t.Parallel()
	for name, items := range map[string][]NamedText{
		"name only": {{name: "T"}}, "text only": {{text: "x"}},
	} {
		if err := checkNamedTexts("Species", items); err == nil {
			t.Errorf("%s: accepted, want BlockInvalidMember", name)
		}
	}
	if err := checkNamedTexts("Species", []NamedText{{name: "T", text: "x"}}); err != nil {
		t.Errorf("a valid trait: %v, want accepted", err)
	}
}
