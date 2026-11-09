package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/character-creation.md section 3 (EquipmentOptionLetter, Q-11) - the letters a, b and c are valid, anything else is not.
// Path: per case.
func TestEquipmentOptionLetter(t *testing.T) {
	t.Parallel()
	for letter, want := range map[character.EquipmentOptionLetter]string{character.OptionA: "a", character.OptionB: "b", character.OptionC: "c", 0: "unknown", 4: "unknown"} {
		if letter.String() != want || letter.Valid() != (want != "unknown") {
			t.Errorf("letter %d: String %q, Valid %v", letter, letter.String(), letter.Valid())
		}
	}
}

// Requirement: doc/specs/character-creation.md FR-088, FR-089 - a trinket roll is 1 to 100; 0 and 101 are validation errors; 100 is kept as 100.
// Path: per case.
func TestTrinketRoll(t *testing.T) {
	t.Parallel()
	for _, n := range []int{1, 50, 100} {
		if r, err := character.NewTrinketRoll(n); err != nil || r.Value() != n || r.IsZero() {
			t.Errorf("NewTrinketRoll(%d) = %v, %v", n, r, err)
		}
	}
	for _, n := range []int{0, 101, -1} {
		if _, err := character.NewTrinketRoll(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewTrinketRoll(%d) error = %v", n, err)
		}
	}
	if !(character.TrinketRoll{}).IsZero() {
		t.Error("the zero TrinketRoll is not zero")
	}
}

// Requirement: doc/specs/character-creation.md FR-093, FR-106 - SpeciesGrantSelections copies its map in and out, compares by value with
// Equal, reports none as none, and rejects an invalid skill, feat id or ability.
// Path: per case.
func TestSpeciesGrantSelections(t *testing.T) {
	t.Parallel()
	in := map[string]ability.Ability{"Elven Lineage": ability.Wisdom}
	g, err := character.NewSpeciesGrantSelections(content.SkillInsight, "alert", in)
	if err != nil {
		t.Fatal(err)
	}
	in["Elven Lineage"] = ability.Strength
	out := g.Abilities()
	out["Elven Lineage"] = ability.Strength
	if g.Abilities()["Elven Lineage"] != ability.Wisdom {
		t.Error("a caller changed the stored map")
	}
	if s, ok := g.Skill(); !ok || s != content.SkillInsight {
		t.Errorf("Skill = %v, %v", s, ok)
	}
	if f, ok := g.Feat(); !ok || f != "alert" {
		t.Errorf("Feat = %v, %v", f, ok)
	}
	same := must(character.NewSpeciesGrantSelections(content.SkillInsight, "alert", map[string]ability.Ability{"Elven Lineage": ability.Wisdom}))
	other := must(character.NewSpeciesGrantSelections(content.SkillInsight, "alert", nil))
	if !g.Equal(same) || g.Equal(other) || g.Equal(must(character.NewSpeciesGrantSelections(content.SkillArcana, "alert", g.Abilities()))) || g.Equal(must(character.NewSpeciesGrantSelections(content.SkillInsight, "skilled", g.Abilities()))) {
		t.Error("Equal compares the wrong fields")
	}
	none := character.SpeciesGrantSelections{}
	if _, ok := none.Skill(); ok {
		t.Error("the zero value has a skill")
	}
	if _, ok := none.Feat(); ok {
		t.Error("the zero value has a feat")
	}
	for name, args := range map[string]func() error{
		"skill": func() error { _, e := character.NewSpeciesGrantSelections(content.Skill(99), "", nil); return e },
		"feat":  func() error { _, e := character.NewSpeciesGrantSelections(0, "Bad Id", nil); return e },
		"ability": func() error {
			_, e := character.NewSpeciesGrantSelections(0, "", map[string]ability.Ability{"x": 0})
			return e
		},
	} {
		if err := args(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("invalid %s error = %v", name, err)
		}
	}
}

// Requirement: doc/specs/character-creation.md FR-093 - SpeciesGrantSelections.Equal is symmetric: it tells selections apart whichever of two
// skills or feats is the greater, and a negative skill is invalid.
// Path: unhappy (boundary).
func TestSpeciesGrantSelectionsEqualInBothDirections(t *testing.T) {
	t.Parallel()
	a := must(character.NewSpeciesGrantSelections(content.SkillArcana, "alert", nil))
	b := must(character.NewSpeciesGrantSelections(content.SkillInsight, "skilled", nil))
	if a.Equal(b) || b.Equal(a) || !a.Equal(a) {
		t.Error("Equal is not symmetric or not reflexive")
	}
	onlySkill := must(character.NewSpeciesGrantSelections(content.SkillInsight, "alert", nil))
	if a.Equal(onlySkill) || onlySkill.Equal(a) {
		t.Error("Equal ignores the skill")
	}
	onlyFeat := must(character.NewSpeciesGrantSelections(content.SkillArcana, "skilled", nil))
	if a.Equal(onlyFeat) || onlyFeat.Equal(a) {
		t.Error("Equal ignores the feat")
	}
	if _, err := character.NewSpeciesGrantSelections(content.Skill(-1), "", nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative skill error = %v", err)
	}
}
