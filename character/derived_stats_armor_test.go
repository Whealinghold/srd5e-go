package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func trainingOf(t *testing.T, cats ...content.ArmorCategory) character.ArmorTraining {
	t.Helper()
	return must(character.NewArmorTraining(cats...))
}

// Requirement: doc/specs/derived-stats.md FR-030, FR-034 to FR-040, FR-054,
// FR-080, FR-081 - the Armor Class cases of section 4.7 (SRD 5.2.1, "Armor", p.
// 92; "Unarmored Defense", pp. 29, 50, 69; "Mage Armor", p. 145; "Barkskin", p.
// 112).
// Path: per case.
func TestArmorClassCases(t *testing.T) {
	t.Parallel()
	type row struct {
		name, path         string
		armor, shield      string // "" none
		train              []content.ArmorCategory
		dex, con, wis, cha int // ability modifiers
		alts               []string
		chosen             character.ArmorClassBasisChoice
		flats              []int
		floor              int
		want               int
		gaps               []content.ArmorCategory
		basis              string // "standard", "armored" or an alternative's name
	}
	heavy, light, medium, shield := content.ArmorHeavy, content.ArmorLight, content.ArmorMedium, content.ArmorShield
	rows := []row{
		{name: "unarmored, positive Dex", path: "happy", dex: 3, want: 13, basis: "standard"},
		{name: "Heavy without training keeps its AC", path: "unhappy (rule interaction)", armor: "chain", train: []content.ArmorCategory{light, medium}, want: 16, gaps: []content.ArmorCategory{heavy}, basis: "armored"},
		{name: "Shield trained through a multiclass", path: "unhappy (rule interaction)", shield: "shield", train: []content.ArmorCategory{shield}, dex: 2, want: 14, basis: "standard"},
		{name: "plate with trained Shield", path: "happy", armor: "plate", shield: "shield", train: []content.ArmorCategory{heavy, shield}, want: 20, basis: "armored"},
		{name: "plate, untrained Shield", path: "unhappy (rule interaction)", armor: "plate", shield: "shield", train: []content.ArmorCategory{heavy}, want: 18, gaps: []content.ArmorCategory{shield}, basis: "armored"},
		{name: "unarmored with trained Shield", path: "happy", shield: "shield", train: []content.ArmorCategory{shield}, dex: 2, want: 14, basis: "standard"},
		{name: "unarmored, untrained Shield", path: "unhappy (rule interaction)", shield: "shield", dex: 2, want: 12, gaps: []content.ArmorCategory{shield}, basis: "standard"},
		{name: "Barbarian Unarmored Defense without Shield", path: "happy", train: []content.ArmorCategory{shield}, dex: 2, con: 3, alts: []string{"barbarian"}, want: 15, basis: "barbarian"},
		{name: "Barbarian Unarmored Defense with Shield", path: "happy", shield: "shield", train: []content.ArmorCategory{shield}, dex: 2, con: 3, alts: []string{"barbarian"}, want: 17, basis: "barbarian"},
		{name: "Barbarian wearing armor", path: "unhappy (rule interaction)", armor: "hide", train: []content.ArmorCategory{medium}, dex: 2, con: 3, alts: []string{"barbarian"}, want: 14, basis: "armored"},
		{name: "Monk Unarmored Defense without Shield", path: "happy", dex: 3, wis: 3, alts: []string{"monk"}, want: 16, basis: "monk"},
		{name: "Monk with a Shield falls back to Standard", path: "happy", shield: "shield", dex: 3, wis: 3, alts: []string{"monk"}, want: 13, gaps: []content.ArmorCategory{shield}, basis: "standard"},
		{name: "tie resolves to Standard", path: "unhappy (boundary)", train: []content.ArmorCategory{shield}, dex: 2, alts: []string{"barbarian"}, chosen: nil, want: 12, basis: "standard"},
		{name: "Sorcerer Draconic unarmored", path: "happy", dex: 2, cha: 3, alts: []string{"sorcerer"}, want: 15, basis: "sorcerer"},
		{name: "Barbarian 1 / Monk 1, choose Monk", path: "unhappy (rule interaction)", train: []content.ArmorCategory{shield}, dex: 2, con: 4, wis: 1, alts: []string{"barbarian", "monk"}, chosen: character.ChooseAlternative{ID: content.MustFeatureID("monk_unarmored_defense")}, want: 13, basis: "monk"},
		{name: "Barbarian 1 / Monk 1, choose Standard", path: "unhappy (rule interaction)", train: []content.ArmorCategory{shield}, dex: 2, con: 4, wis: 1, alts: []string{"barbarian", "monk"}, chosen: character.ChooseStandard{}, want: 12, basis: "standard"},
		{name: "choose the earlier-sorting alternative listed second", path: "unhappy (determinism)", train: []content.ArmorCategory{shield}, dex: 2, con: 4, wis: 1, alts: []string{"monk", "barbarian"}, chosen: character.ChooseAlternative{ID: content.MustFeatureID("barbarian_unarmored_defense")}, want: 16, basis: "barbarian"},
		{name: "two bases none chosen, Barbarian higher", path: "unhappy (rule interaction)", dex: 2, con: 4, wis: 1, alts: []string{"barbarian", "monk"}, want: 16, basis: "barbarian"},
		{name: "two bases none chosen, Monk higher", path: "unhappy (rule interaction)", dex: 2, con: 1, wis: 5, alts: []string{"barbarian", "monk"}, want: 17, basis: "monk"},
		{name: "equal alternatives resolve in command order", path: "unhappy (determinism)", dex: 2, con: 2, wis: 2, alts: []string{"monk", "barbarian"}, want: 14, basis: "monk"},
		{name: "chosen Armored with armor", path: "happy", armor: "leather", train: []content.ArmorCategory{light}, dex: 2, chosen: character.ChooseArmored{}, want: 13, basis: "armored"},
		{name: "flat bonus added", path: "happy", armor: "leather", train: []content.ArmorCategory{light}, dex: 2, flats: []int{1}, want: 14, basis: "armored"},
		{name: "Mage Armor base 13", path: "happy", dex: 2, alts: []string{"mage"}, want: 15, basis: "mage"},
		{name: "Mage Armor and Shield of Faith, floor 17 adds nothing", path: "unhappy (rule interaction)", dex: 2, alts: []string{"mage"}, flats: []int{2}, floor: 17, want: 17, basis: "mage"},
		{name: "Mage Armor and Shield of Faith, floor 18", path: "unhappy (rule interaction)", dex: 2, alts: []string{"mage"}, flats: []int{2}, floor: 18, want: 18, basis: "mage"},
		{name: "Barkskin floor on low AC", path: "unhappy (boundary)", armor: "leather", train: []content.ArmorCategory{light}, floor: 17, want: 17, basis: "armored"},
		{name: "Barkskin floor on high AC", path: "unhappy (boundary)", armor: "plate", train: []content.ArmorCategory{heavy}, floor: 17, want: 18, basis: "armored"},
		{name: "floor equal to the subtotal", path: "unhappy (boundary)", armor: "plate", train: []content.ArmorCategory{heavy}, floor: 18, want: 18, basis: "armored"},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			armors := map[string]content.Armor{"chain": chainMail(t), "plate": plateArmor(t), "hide": hideArmor(t), "leather": leatherArmor(t), "shield": shieldOf(t)}
			in := base(t)
			in.AbilityScores = scoresOf(t, 10, scoreForModifier(r.dex), scoreForModifier(r.con), 10, scoreForModifier(r.wis), scoreForModifier(r.cha))
			in.Armor, in.Shield = armors[r.armor], armors[r.shield]
			in.MulticlassArmorTraining = trainingOf(t, r.train...)
			for _, a := range r.alts {
				in.AlternativeBases = append(in.AlternativeBases, basisOf(t, a))
			}
			in.ChosenBasis = r.chosen
			for _, f := range r.flats {
				in.FlatArmorClassBonuses = append(in.FlatArmorClassBonuses, itemBonus(t, f))
			}
			in.ArmorClassFloor = r.floor
			s := calculate(t, in)
			if s.ArmorClass() != d20.ArmorClass(r.want) {
				t.Errorf("Armor Class = %d, want %d; terms %+v", s.ArmorClass(), r.want, s.ArmorClassBreakdown().Terms())
			}
			if !slices.Equal(s.ArmorTrainingGaps(), r.gaps) {
				t.Errorf("gaps = %v, want %v", s.ArmorTrainingGaps(), r.gaps)
			}
			for _, term := range s.ArmorClassBreakdown().Terms() {
				if term.Source == character.FloorTerm && term.Modifier.Value() <= 0 {
					t.Errorf("floor term = %d, want a positive top-up only", term.Modifier.Value())
				}
			}
			switch b := s.ArmorClassBasis().(type) {
			case character.StandardArmorClassBasis:
				if r.basis != "standard" {
					t.Errorf("basis = standard, want %s", r.basis)
				}
			case character.ArmoredArmorClassBasis:
				if r.basis != "armored" || b.Armor != in.Armor {
					t.Errorf("basis = armored %v, want %s", b.Armor.ID(), r.basis)
				}
			case character.AbilityBasedArmorClassBasis:
				if got := basisOf(t, r.basis); b.ID() != got.ID() {
					t.Errorf("basis = %v, want %v", b.ID(), got.ID())
				}
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-041 - the Armor Class terms in the
// order Standard base, Dexterity, Shield bonus, flat bonuses, floor; the Shield
// bonus is read from the entry's rule, not a constant (FR-034).
// Path: happy.
func TestArmorClassTermsAndShieldFromContent(t *testing.T) {
	t.Parallel()
	in := base(t)
	in.AbilityScores = scoresOf(t, 10, 14, 10, 10, 10, 10)
	in.Shield = armorOf(t, "tower_shield", content.ArmorShield, must(content.NewBonusRule(3)), 0, false, 6, gp(10))
	in.MulticlassArmorTraining = trainingOf(t, content.ArmorShield)
	in.FlatArmorClassBonuses = []character.FlatBonus{itemBonus(t, 1)}
	in.ArmorClassFloor = 20
	terms := calculate(t, in).ArmorClassBreakdown().Terms()
	var got []character.DerivedStatSource
	for _, term := range terms {
		got = append(got, term.Source)
	}
	want := []character.DerivedStatSource{character.BaseValueTerm, character.AbilityModifierTerm, character.ShieldBonusTerm, character.FlatBonusTerm, character.FloorTerm}
	values := []int{10, 2, 3, 1, 4}
	if !slices.Equal(got, want) {
		t.Fatalf("sources = %v, want %v", got, want)
	}
	for i, term := range terms {
		if term.Modifier.Value() != values[i] {
			t.Errorf("term %d (%v) = %d, want %d", i, term.Source, term.Modifier.Value(), values[i])
		}
	}
	if terms[1].Ability != ability.Dexterity {
		t.Errorf("Dexterity term names %v", terms[1].Ability)
	}
}

// Requirement: doc/specs/derived-stats.md FR-041 - an Armored base shows the
// listed base and a Dexterity term equal to dexterityContribution, omitted when
// it is 0 (equipment-rules.md FR-004).
// Path: per case.
func TestArmoredTerms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		armor func(*testing.T) content.Armor
		dex   int
		terms []int // listed base, then the Dexterity contribution when present
	}{
		{"full Dexterity", leatherArmor, 3, []int{11, 3}},
		{"capped Dexterity", hideArmor, 4, []int{12, 2}},
		{"no Dexterity", chainMail, 4, []int{16}},
		{"full Dexterity, zero modifier", leatherArmor, 0, []int{11}},
		{"capped, negative modifier", hideArmor, -1, []int{12, -1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base(t)
			in.AbilityScores = scoresOf(t, 10, scoreForModifier(c.dex), 10, 10, 10, 10)
			in.Armor = c.armor(t)
			terms := calculate(t, in).ArmorClassBreakdown().Terms()
			var got []int
			for _, term := range terms {
				got = append(got, term.Modifier.Value())
			}
			if !slices.Equal(got, c.terms) || terms[0].Source != character.ArmorBaseTerm {
				t.Errorf("terms = %v (first %v), want %v", got, terms[0].Source, c.terms)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-037, FR-054, FR-076 - a chosen basis
// that cannot be used is a result with the choice and a reason; an unsupplied
// alternative is reported before applicability, even with armor worn.
// Path: per case.
func TestChosenBasisNotApplicable(t *testing.T) {
	t.Parallel()
	barb := character.ChooseAlternative{ID: content.MustFeatureID("barbarian_unarmored_defense")}
	monk := character.ChooseAlternative{ID: content.MustFeatureID("monk_unarmored_defense")}
	cases := []struct {
		name, path    string
		armor, shield string
		alts          []string
		chosen        character.ArmorClassBasisChoice
		want          character.ArmorClassBasisRejection
	}{
		{"chosen alternative, armor worn", "unhappy (state error)", "leather", "", []string{"barbarian"}, barb, character.BasisNotApplicable},
		{"chosen Monk, Shield wielded", "unhappy (state error)", "", "shield", []string{"monk"}, monk, character.BasisNotApplicable},
		{"chosen alternative not supplied", "unhappy (state error)", "", "", nil, barb, character.BasisAlternativeNotSupplied},
		{"chosen alternative not supplied, a later-sorting one is", "unhappy (state error)", "", "", []string{"monk"}, barb, character.BasisAlternativeNotSupplied},
		{"chosen alternative not supplied, an earlier-sorting one is", "unhappy (state error)", "", "", []string{"barbarian"}, monk, character.BasisAlternativeNotSupplied},
		{"not supplied, checked before applicability", "unhappy (state error)", "leather", "", nil, barb, character.BasisAlternativeNotSupplied},
		{"chosen Standard with armor", "unhappy (state error)", "leather", "", nil, character.ChooseStandard{}, character.BasisNotApplicable},
		{"chosen Armored without armor", "unhappy (state error)", "", "", nil, character.ChooseArmored{}, character.BasisNotApplicable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			armors := map[string]content.Armor{"leather": leatherArmor(t), "shield": shieldOf(t)}
			in := base(t)
			in.Armor, in.Shield = armors[c.armor], armors[c.shield]
			for _, a := range c.alts {
				in.AlternativeBases = append(in.AlternativeBases, basisOf(t, a))
			}
			in.ChosenBasis = c.chosen
			got, err := character.DerivedStatsCalculator{}.Calculate(in)
			if err != nil {
				t.Fatal(err)
			}
			if want := (character.DerivedStatsBasisNotApplicable{Choice: c.chosen, Reason: c.want}); got != want {
				t.Errorf("Calculate = %#v, want %#v", got, want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-053, FR-068 - an armor whose rule
// does not suit its slot is a mismatch naming the entry's key and the slot, the
// body armor first.
// Path: per case.
func TestSlotMismatch(t *testing.T) {
	t.Parallel()
	sh, leather := shieldOf(t), leatherArmor(t)
	cases := []struct {
		name, path    string
		armor, shield content.Armor
		want          character.DerivedStatsSlotMismatch
	}{
		{"Shield as armorId", "unhappy (invalid input)", sh, content.Armor{}, character.DerivedStatsSlotMismatch{Key: sh.Key(), Expected: character.BodyArmorSlot}},
		{"body armor as shieldId", "unhappy (invalid input)", content.Armor{}, leather, character.DerivedStatsSlotMismatch{Key: leather.Key(), Expected: character.ShieldSlot}},
		{"both mismatch, armor reported", "unhappy (rule interaction)", sh, leather, character.DerivedStatsSlotMismatch{Key: sh.Key(), Expected: character.BodyArmorSlot}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base(t)
			in.Armor, in.Shield = c.armor, c.shield
			got, err := character.DerivedStatsCalculator{}.Calculate(in)
			if err != nil || got != c.want {
				t.Errorf("Calculate = %#v, %v, want %#v", got, err, c.want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-077 - untrained body armor gives one
// Disadvantage when Strength or Dexterity is involved, never a breakdown term;
// an untrained Shield alone gives none; the Armor Class is unchanged (SRD 5.2.1,
// "Armor Training", p. 177).
// Path: per case.
func TestArmorTrainingCircumstances(t *testing.T) {
	t.Parallel()
	str, dex, con := []ability.Ability{ability.Strength}, []ability.Ability{ability.Dexterity}, []ability.Ability{ability.Constitution}
	cases := []struct {
		name, path    string
		armor, shield string
		train         []content.ArmorCategory
		involved      []ability.Ability
		want          int
	}{
		{"untrained Heavy, Strength", "happy", "chain", "", []content.ArmorCategory{content.ArmorLight, content.ArmorMedium}, str, 1},
		{"untrained Medium, Dexterity", "unhappy (rule interaction)", "hide", "", []content.ArmorCategory{content.ArmorLight}, dex, 1},
		{"untrained, other ability", "unhappy (rule interaction)", "chain", "", []content.ArmorCategory{content.ArmorLight}, con, 0},
		{"trained", "unhappy (rule interaction)", "chain", "", []content.ArmorCategory{content.ArmorHeavy}, str, 0},
		{"untrained Shield alone", "unhappy (rule interaction)", "", "shield", nil, []ability.Ability{ability.Strength, ability.Dexterity}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			armors := map[string]content.Armor{"chain": chainMail(t), "hide": hideArmor(t), "shield": shieldOf(t)}
			in := base(t)
			in.Armor, in.Shield = armors[c.armor], armors[c.shield]
			in.MulticlassArmorTraining = trainingOf(t, c.train...)
			s := calculate(t, in)
			got := s.ArmorTrainingCircumstances(c.involved)
			if len(got) != c.want || (c.want == 1 && got[0] != modifier.Disadvantage{}) {
				t.Errorf("circumstances = %v, want %d Disadvantage", got, c.want)
			}
			for _, term := range s.ArmorClassBreakdown().Terms() {
				if term.Source == character.RollModeTerm {
					t.Error("a circumstance appears as a breakdown term")
				}
			}
		})
	}
	// AC unchanged: Chain Mail untrained, Dexterity +5.
	in := base(t)
	in.AbilityScores = scoresOf(t, 10, 20, 10, 10, 10, 10)
	in.Armor = chainMail(t)
	in.MulticlassArmorTraining = trainingOf(t, content.ArmorLight)
	s := calculate(t, in)
	if s.ArmorClass() != 16 || len(s.ArmorTrainingCircumstances(dex)) != 1 {
		t.Errorf("AC %d, circumstances %v", s.ArmorClass(), s.ArmorTrainingCircumstances(dex))
	}
}

// Requirement: doc/specs/derived-stats.md FR-040 - training is the union of the
// initial class's armorTraining and the multiclass grants, converted by name.
// Path: unhappy (rule interaction).
func TestArmorTrainingUnion(t *testing.T) {
	t.Parallel()
	in := base(t)
	in.InitialClass = fighterClass(t)
	in.ClassLevels = levels(cl("fighter", 1))
	in.Armor, in.Shield = plateArmor(t), shieldOf(t)
	s := calculate(t, in)
	if len(s.ArmorTrainingGaps()) != 0 || s.ArmorClass() != 20 {
		t.Errorf("Fighter in plate with a Shield: AC %d, gaps %v", s.ArmorClass(), s.ArmorTrainingGaps())
	}
	in.InitialClass = classFixture(t, "cleric", "D8", []string{"wisdom", "charisma"}, []string{"Light", "Medium", "Shield"})
	in.ClassLevels = levels(cl("cleric", 1))
	s = calculate(t, in)
	if want := []content.ArmorCategory{content.ArmorHeavy}; !slices.Equal(s.ArmorTrainingGaps(), want) || s.ArmorClass() != 20 {
		t.Errorf("Cleric in plate: AC %d, gaps %v, want gaps %v", s.ArmorClass(), s.ArmorTrainingGaps(), want)
	}
	in.MulticlassArmorTraining = trainingOf(t, content.ArmorHeavy)
	if s = calculate(t, in); len(s.ArmorTrainingGaps()) != 0 {
		t.Errorf("multiclass Heavy training left gaps %v", s.ArmorTrainingGaps())
	}
}
