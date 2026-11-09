package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func rogueRules() character.WeaponProficiencies {
	return character.NewWeaponProficiencies(
		character.CategoryProficiency(content.WeaponSimple),
		character.MustCategoryWithAnyPropertyProficiency(content.WeaponMartial, []content.WeaponProperty{content.WeaponPropertyFinesse, content.WeaponPropertyLight}),
	)
}

func monkRules() character.WeaponProficiencies {
	return character.NewWeaponProficiencies(
		character.CategoryProficiency(content.WeaponSimple),
		character.MustCategoryWithAnyPropertyProficiency(content.WeaponMartial, []content.WeaponProperty{content.WeaponPropertyLight}),
	)
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-007 to FR-010, FR-012, FR-013,
// table 4.1a - StatusFor is proficient exactly when a rule matches; the property rules read
// property names only (SRD 5.2.1, "Weapon Proficiency", p. 89; "Core Rogue Traits", p. 61).
// Path: per case.
func TestWeaponProficienciesStatusFor(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	simpleOnly := character.NewWeaponProficiencies(character.CategoryProficiency(content.WeaponSimple))
	martialOnly := character.NewWeaponProficiencies(character.CategoryProficiency(content.WeaponMartial))
	cases := []struct {
		name, requirement, path string
		rules                   character.WeaponProficiencies
		weapon                  string
		want                    character.WeaponProficiencyStatus
	}{
		{"Simple covers Club", "FR-008", "happy", simpleOnly, "club", character.WeaponProficient},
		{"Simple excludes Longsword", "FR-008", "unhappy (boundary)", simpleOnly, "longsword", character.WeaponNotProficient},
		{"Martial covers Longsword", "FR-008", "happy", martialOnly, "longsword", character.WeaponProficient},
		{"Rogue: Simple Dart", "FR-010", "happy", rogueRules(), "dart", character.WeaponProficient},
		{"Rogue: Rapier", "FR-009", "unhappy (boundary)", rogueRules(), "rapier", character.WeaponProficient},
		{"Rogue: Scimitar (both)", "FR-009", "unhappy (boundary)", rogueRules(), "scimitar", character.WeaponProficient},
		{"Rogue: Greatsword", "FR-009", "unhappy (rule interaction)", rogueRules(), "greatsword", character.WeaponNotProficient},
		{"Rogue: Whip", "FR-009", "unhappy (boundary)", rogueRules(), "whip", character.WeaponProficient},
		{"Monk: Whip", "FR-009", "unhappy (boundary)", monkRules(), "whip", character.WeaponNotProficient},
		{"Monk: Shortsword", "FR-009", "happy", monkRules(), "shortsword", character.WeaponProficient},
		{"Monk: Longbow", "FR-009", "unhappy (rule interaction)", monkRules(), "longbow", character.WeaponNotProficient},
		{"No rule", "FR-010", "unhappy (empty case)", character.WeaponProficiencies{}, "dagger", character.WeaponNotProficient},
		{"Monster", "FR-013", "happy", character.NewWeaponProficiencies(character.AnyStatBlockWeaponProficiency{}), "pike", character.WeaponProficient},
		{"Thrown range 20/60", "FR-012", "happy", character.NewWeaponProficiencies(character.MustCategoryWithAnyPropertyProficiency(content.WeaponSimple, []content.WeaponProperty{content.WeaponPropertyThrown})), "dagger", character.WeaponProficient},
		{"Thrown range 30/120", "FR-012", "unhappy (rule interaction)", character.NewWeaponProficiencies(character.MustCategoryWithAnyPropertyProficiency(content.WeaponSimple, []content.WeaponProperty{content.WeaponPropertyThrown})), "javelin", character.WeaponProficient},
		{"Versatile damage differs", "FR-012", "unhappy (rule interaction)", character.NewWeaponProficiencies(character.MustCategoryWithAnyPropertyProficiency(content.WeaponMartial, []content.WeaponProperty{content.WeaponPropertyVersatile})), "longsword", character.WeaponProficient},
		{"Versatile on a Simple weapon", "FR-012", "unhappy (boundary)", character.NewWeaponProficiencies(character.MustCategoryWithAnyPropertyProficiency(content.WeaponMartial, []content.WeaponProperty{content.WeaponPropertyVersatile})), "spear", character.WeaponNotProficient},
		{"nil rule ignored", "FR-010", "unhappy (empty case)", character.NewWeaponProficiencies(nil), "dagger", character.WeaponNotProficient},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.rules.StatusFor(ws[c.weapon]); got != c.want {
				t.Errorf("StatusFor(%s) = %v, want %v [%s, %s]", c.weapon, got, c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-011 - an empty property set or an
// invalid category is rejected with an input error; Must panics with it.
// Path: unhappy (invalid input).
func TestCategoryWithAnyPropertyProficiencyRejectsInvalid(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		cat   content.WeaponCategory
		props []content.WeaponProperty
	}{"empty set": {content.WeaponMartial, nil}, "empty slice": {content.WeaponMartial, []content.WeaponProperty{}}, "bad category": {0, []content.WeaponProperty{content.WeaponPropertyLight}}} {
		_, err := character.NewCategoryWithAnyPropertyProficiency(c.cat, c.props)
		wantInvalid(t, err, name)
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("%s: Must did not panic", name)
				}
			}()
			character.MustCategoryWithAnyPropertyProficiency(c.cat, c.props)
		}()
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-105, FR-011 - the property rule copies its set on
// construction and on read, and compares by value.
// Path: unhappy (immutability).
func TestCategoryWithAnyPropertyProficiencyCopiesItsSet(t *testing.T) {
	t.Parallel()
	in := []content.WeaponProperty{content.WeaponPropertyLight}
	r := character.MustCategoryWithAnyPropertyProficiency(content.WeaponMartial, in)
	in[0] = content.WeaponPropertyHeavy
	out := r.Properties()
	out[0] = content.WeaponPropertyHeavy
	if got := r.Properties(); len(got) != 1 || got[0] != content.WeaponPropertyLight || r.Category() != content.WeaponMartial {
		t.Errorf("rule changed: %v", got)
	}
	same := character.MustCategoryWithAnyPropertyProficiency(content.WeaponMartial, []content.WeaponProperty{content.WeaponPropertyLight})
	other := character.MustCategoryWithAnyPropertyProficiency(content.WeaponSimple, []content.WeaponProperty{content.WeaponPropertyLight})
	if !r.Equal(same) || r.Equal(other) {
		t.Error("Equal does not compare by value")
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-120, FR-010 - one rule per element of the class
// data; an empty slice gives no rule.
// Path: happy.
func TestWeaponProficienciesFromClassData(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	rogue := []content.WeaponProficiency{
		must(content.NewWeaponProficiency(content.WeaponTrainingSimple, nil)),
		must(content.NewWeaponProficiency(content.WeaponTrainingMartial, []content.WeaponProperty{content.WeaponPropertyFinesse, content.WeaponPropertyLight})),
	}
	got := character.WeaponProficienciesFromClassData(rogue)
	for w, want := range map[string]character.WeaponProficiencyStatus{
		"club": character.WeaponProficient, "rapier": character.WeaponProficient, "greatsword": character.WeaponNotProficient, "dart": character.WeaponProficient,
	} {
		if s := got.StatusFor(ws[w]); s != want {
			t.Errorf("Rogue %s = %v, want %v", w, s, want)
		}
	}
	martial := character.WeaponProficienciesFromClassData([]content.WeaponProficiency{must(content.NewWeaponProficiency(content.WeaponTrainingMartial, nil))})
	if martial.StatusFor(ws["greatsword"]) != character.WeaponProficient || martial.StatusFor(ws["club"]) != character.WeaponNotProficient {
		t.Error("Martial training must cover Martial weapons only")
	}
	if character.WeaponProficienciesFromClassData(nil).StatusFor(ws["club"]) != character.WeaponNotProficient {
		t.Error("an empty slice must give no rule")
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-007 - the status names are the short names.
// Path: happy.
func TestWeaponProficiencyStatusString(t *testing.T) {
	t.Parallel()
	for s, want := range map[character.WeaponProficiencyStatus]string{character.WeaponProficient: "proficient", character.WeaponNotProficient: "notProficient", 0: "unknown"} {
		if s.String() != want {
			t.Errorf("%d = %q, want %q", int(s), s.String(), want)
		}
	}
}
