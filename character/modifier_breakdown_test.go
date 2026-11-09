package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func term(v int, s character.DerivedStatSource) character.BreakdownTerm {
	return character.BreakdownTerm{Modifier: modifier.New(v), Source: s}
}

func abilityTerm(v int, a ability.Ability) character.BreakdownTerm {
	return character.BreakdownTerm{Modifier: modifier.New(v), Source: character.AbilityModifierTerm, Ability: a}
}

func flatTerm(v int) character.BreakdownTerm {
	return character.BreakdownTerm{Modifier: modifier.New(v), Source: character.FlatBonusTerm,
		Origin: character.SpellFlatBonusSource{Spell: content.MustSpellID("shield_of_faith")}}
}

// Requirement: doc/specs/derived-stats.md FR-010, FR-056, FR-067 - the
// constructor returns an error matching ErrInvalidInput for each invalid term
// combination and allows several ability terms (SRD 5.2.1, "The Bonus Doesn't
// Stack", p. 8).
// Path: per case.
func TestNewModifierBreakdown(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		terms      []character.BreakdownTerm
		problems   int // joined errors expected; 0 means valid
	}{
		{"valid save", "happy", []character.BreakdownTerm{abilityTerm(2, ability.Constitution), term(2, character.ProficiencyBonusTerm)}, 0},
		{"empty", "unhappy (empty and minimal)", nil, 0},
		{"several ability terms", "unhappy (rule interaction)", []character.BreakdownTerm{term(10, character.BaseValueTerm), abilityTerm(2, ability.Dexterity), abilityTerm(3, ability.Constitution)}, 0},
		{"flat bonus with origin", "happy", []character.BreakdownTerm{abilityTerm(1, ability.Dexterity), flatTerm(1)}, 0},
		{"proficiency and expertise", "unhappy (rule interaction)", []character.BreakdownTerm{term(2, character.ProficiencyBonusTerm), term(4, character.ExpertiseTerm)}, 1},
		{"two proficiency terms", "unhappy (rule interaction)", []character.BreakdownTerm{term(2, character.ProficiencyBonusTerm), term(2, character.ProficiencyBonusTerm)}, 1},
		{"ability term without ability", "unhappy (invalid input)", []character.BreakdownTerm{term(2, character.AbilityModifierTerm)}, 1},
		{"ability on another term", "unhappy (invalid input)", []character.BreakdownTerm{{Modifier: modifier.New(2), Source: character.BaseValueTerm, Ability: ability.Strength}}, 1},
		{"flat bonus without origin", "unhappy (invalid input)", []character.BreakdownTerm{term(1, character.FlatBonusTerm)}, 1},
		{"origin on another term", "unhappy (invalid input)", []character.BreakdownTerm{{Modifier: modifier.New(2), Source: character.ShieldBonusTerm, Origin: character.SpellFlatBonusSource{}}}, 1},
		{"invalid source", "unhappy (invalid input)", []character.BreakdownTerm{term(1, 0)}, 1},
		{"every violation", "unhappy (invalid input)", []character.BreakdownTerm{term(2, character.ProficiencyBonusTerm), term(4, character.ExpertiseTerm), term(1, character.AbilityModifierTerm), term(1, character.FlatBonusTerm)}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b, err := character.NewModifierBreakdown(c.terms...)
			if c.problems == 0 {
				if err != nil || len(b.Terms()) != len(c.terms) {
					t.Fatalf("NewModifierBreakdown = %v, %v", b, err)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || len(joined(err)) != c.problems {
				t.Errorf("error = %v (%d joined), want %d problems matching ErrInvalidInput", err, len(joined(err)), c.problems)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-056 - the breakdown copies its
// terms in and out, Total is their sum, terms keep their order.
// Path: unhappy (determinism, immutability).
func TestModifierBreakdownCopiesAndTotal(t *testing.T) {
	t.Parallel()
	in := []character.BreakdownTerm{abilityTerm(3, ability.Strength), term(-1, character.RollModeTerm), term(5, character.BaseValueTerm)}
	b := must(character.NewModifierBreakdown(in...))
	in[0].Modifier = modifier.New(100)
	out := b.Terms()
	out[1].Modifier = modifier.New(100)
	if b.Total().Value() != 7 || !slices.Equal(b.Terms(), []character.BreakdownTerm{abilityTerm(3, ability.Strength), term(-1, character.RollModeTerm), term(5, character.BaseValueTerm)}) {
		t.Errorf("breakdown changed: total %d terms %v", b.Total().Value(), b.Terms())
	}
	if !b.Equal(must(character.NewModifierBreakdown(b.Terms()...))) || b.Equal(character.ModifierBreakdown{}) {
		t.Error("Equal wrong")
	}
	if (character.ModifierBreakdown{}).Total().Value() != 0 {
		t.Error("empty total is not 0")
	}
}

// Requirement: doc/specs/derived-stats.md FR-066 - ToRollModifiers maps a save or
// skill breakdown and rejects the others with every problem, in enum order
// (SRD 5.2.1, "D20 Tests", "Add Modifiers", p. 6).
// Path: per case.
func TestToRollModifiers(t *testing.T) {
	t.Parallel()
	const (
		missing  = character.BreakdownMissingAbilityTerm
		multiple = character.BreakdownMultipleAbilityTerms
		unmapped = character.BreakdownUnmappableSource
	)
	cases := []struct {
		name, path string
		terms      []character.BreakdownTerm
		want       []character.BreakdownMappingProblem
	}{
		{"saving throw, proficient", "happy", []character.BreakdownTerm{abilityTerm(2, ability.Constitution), term(2, character.ProficiencyBonusTerm)}, nil},
		{"skill, Expertise", "unhappy (rule interaction)", []character.BreakdownTerm{abilityTerm(3, ability.Strength), term(4, character.ExpertiseTerm)}, nil},
		{"Passive Perception", "unhappy (invalid input)", []character.BreakdownTerm{term(10, character.BaseValueTerm), abilityTerm(2, ability.Wisdom), term(2, character.ProficiencyBonusTerm)}, []character.BreakdownMappingProblem{unmapped}},
		{"two ability terms", "unhappy (rule interaction)", []character.BreakdownTerm{abilityTerm(1, ability.Dexterity), abilityTerm(3, ability.Constitution)}, []character.BreakdownMappingProblem{multiple}},
		{"several problems", "unhappy (invalid input)", []character.BreakdownTerm{term(10, character.BaseValueTerm), abilityTerm(1, ability.Dexterity), abilityTerm(3, ability.Constitution)}, []character.BreakdownMappingProblem{multiple, unmapped}},
		{"Armor Class", "unhappy (invalid input)", []character.BreakdownTerm{term(10, character.BaseValueTerm), abilityTerm(2, ability.Dexterity), term(2, character.ShieldBonusTerm)}, []character.BreakdownMappingProblem{unmapped}},
		{"empty breakdown", "unhappy (empty and minimal)", nil, []character.BreakdownMappingProblem{missing}},
		{"only a base value", "unhappy (invalid input)", []character.BreakdownTerm{term(10, character.BaseValueTerm)}, []character.BreakdownMappingProblem{missing, unmapped}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b := must(character.NewModifierBreakdown(c.terms...))
			rm, err := b.ToRollModifiers()
			if c.want == nil {
				if err != nil || rm.Total().Value() != b.Total().Value() {
					t.Fatalf("ToRollModifiers = %v, %v", rm, err)
				}
				return
			}
			var got []character.BreakdownMappingProblem
			for _, e := range joined(err) {
				var me *character.BreakdownMappingError
				if !errors.As(e, &me) || !errors.Is(e, srderr.ErrInvalidInput) {
					t.Fatalf("error %v is not a *BreakdownMappingError matching ErrInvalidInput", e)
				}
				got = append(got, me.Problem)
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("problems = %v, want %v", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-066 - the mapped slots: the ability
// term is the ability modifier, the one proficiency or expertise term the
// Proficiency Bonus slot, each flat bonus one other, in term order.
// Path: happy.
func TestToRollModifiersSlots(t *testing.T) {
	t.Parallel()
	b := must(character.NewModifierBreakdown(abilityTerm(2, ability.Constitution), term(4, character.ExpertiseTerm), flatTerm(1), flatTerm(3)))
	rm, err := b.ToRollModifiers()
	if err != nil {
		t.Fatal(err)
	}
	prof, ok := rm.ProficiencyBonus()
	if rm.Ability().Value() != 2 || !ok || prof.Value() != 4 || len(rm.Others()) != 2 || rm.Others()[0].Value() != 1 || rm.Others()[1].Value() != 3 {
		t.Errorf("mapped = ability %d prof %v %v others %v", rm.Ability().Value(), prof, ok, rm.Others())
	}
	plain := must(must(character.NewModifierBreakdown(abilityTerm(2, ability.Constitution))).ToRollModifiers())
	if _, ok := plain.ProficiencyBonus(); ok || len(plain.Others()) != 0 {
		t.Error("an unproficient save mapped a proficiency slot")
	}
}

// Requirement: doc/specs/derived-stats.md FR-056 - source and problem names are
// stable and the zero value is unknown.
// Path: happy.
func TestBreakdownNames(t *testing.T) {
	t.Parallel()
	want := map[character.DerivedStatSource]string{
		character.BaseValueTerm: "baseValue", character.AbilityModifierTerm: "abilityModifier", character.ProficiencyBonusTerm: "proficiencyBonus",
		character.ExpertiseTerm: "expertise", character.ArmorBaseTerm: "armorBase", character.ShieldBonusTerm: "shieldBonus",
		character.RollModeTerm: "rollMode", character.FlatBonusTerm: "flatBonus", character.StoredHitPointMaximumTerm: "storedHitPointMaximum",
		character.HitPointMaximumBonusTerm: "hitPointMaximumBonus", character.FloorTerm: "floor", 0: "unknown",
	}
	for s, name := range want {
		if s.String() != name {
			t.Errorf("%d.String() = %q, want %q", s, s.String(), name)
		}
	}
	for p, name := range map[character.BreakdownMappingProblem]string{
		character.BreakdownMissingAbilityTerm: "missingAbilityTerm", character.BreakdownMultipleAbilityTerms: "multipleAbilityTerms",
		character.BreakdownUnmappableSource: "unmappableSource", 0: "unknown",
	} {
		if p.String() != name {
			t.Errorf("%d.String() = %q, want %q", p, p.String(), name)
		}
	}
	e := &character.BreakdownMappingError{Problem: character.BreakdownMissingAbilityTerm}
	if e.Error() == "" || errors.Is(e, &character.BreakdownMappingError{Problem: character.BreakdownMultipleAbilityTerms}) {
		t.Error("error text or Is wrong")
	}
}
