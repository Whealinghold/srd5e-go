package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-spellcasting.md FR-001, FR-011 - NewMonsterSpellcaster holds the trait, the combatant and the Modifier of each of the monster's six scores; a Spellcasting part among the Actions counts as the trait (SRD 5.2.1, "Spellcasting", p. 257).
// Path: happy.
func TestNewMonsterSpellcaster(t *testing.T) {
	t.Parallel()
	m := mscMonster(t, nil)
	c := luMust(combat.NewMonsterSpellcaster(m, mscID()))
	trait, ok := c.Spellcasting()
	want := map[ability.Ability]int{ability.Strength: 0, ability.Dexterity: 0, ability.Constitution: 0, ability.Intelligence: 3, ability.Wisdom: 1, ability.Charisma: 5}
	if !ok || trait.Ability() != ability.Intelligence || !slices.Equal(c.Combatants(), []combat.CombatantID{mscID()}) || c.MonsterID() != m.ID() || c.IsZero() {
		t.Errorf("caster = trait %v (%v), combatant %v, monster %v", trait.Ability(), ok, c.Combatants(), c.MonsterID())
	}
	for a, v := range want {
		if got := c.AbilityModifier(a).Value(); got != v {
			t.Errorf("AbilityModifier(%v) = %d, want %d", a, got, v)
		}
	}
	if got := c.AbilityModifier(0).Value(); got != 0 {
		t.Errorf("AbilityModifier(invalid) = %d, want 0", got)
	}
	if got := c.AbilityModifier(ability.Charisma + 1).Value(); got != 0 {
		t.Errorf("AbilityModifier(past the six) = %d, want 0", got)
	}
	inActions := luMonster(t, func(f *content.MonsterFields) {
		f.Actions = []content.StatBlockPart{mscTraitPart("Spellcasting", mscTraitFields())}
	})
	if _, err := combat.NewMonsterSpellcaster(inActions, mscID()); err != nil {
		t.Errorf("a Spellcasting action: error = %v, want none", err)
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-001 - a monster without a part named Spellcasting that has a structured trait, one with two such parts, one whose Spellcasting is not the first of the stat block, a zero monster and a zero combatant are errors matching srderr.ErrInvalidInput; a caster for parts needs only a monster and a combatant.
// Path: per case.
func TestNewMonsterSpellcasterRefusals(t *testing.T) {
	t.Parallel()
	trait := func(name string) content.StatBlockPart { return mscTraitPart(name, mscTraitFields()) }
	cases := []struct {
		name, requirement, path string
		monster                 content.Monster
		combatant               combat.CombatantID
		parts                   bool // whether the caster for parts accepts it
	}{
		{"no Spellcasting part", "FR-001", "unhappy (missing content)", luMonster(t, nil), mscID(), true},
		{"a structured part with another name", "FR-001", "unhappy (missing content)", luMonster(t, func(f *content.MonsterFields) {
			f.Traits = []content.StatBlockPart{trait("Hellfire Spellcasting")}
		}), mscID(), true},
		{"a part named Spellcasting with no structure", "FR-001", "unhappy (missing content)", luMonster(t, func(f *content.MonsterFields) {
			f.Traits = []content.StatBlockPart{luPart("Spellcasting", content.PartUnlimited{})}
		}), mscID(), true},
		{"two Spellcasting parts", "FR-001", "unhappy (missing content)", luMonster(t, func(f *content.MonsterFields) {
			f.Traits = []content.StatBlockPart{trait("Spellcasting")}
			f.Actions = []content.StatBlockPart{trait("Spellcasting")}
		}), mscID(), true},
		{"not the first Spellcasting", "FR-001", "unhappy (invalid input)", luMonster(t, func(f *content.MonsterFields) {
			other := mscTraitFields()
			other.Ability = ability.Wisdom
			f.Traits = []content.StatBlockPart{mscTraitPart("Hellfire Spellcasting", other), trait("Spellcasting")}
		}), mscID(), true},
		{"zero monster", "FR-001", "unhappy (invalid input)", content.Monster{}, mscID(), false},
		{"zero combatant", "FR-001", "unhappy (invalid input)", mscMonster(t, nil), "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewMonsterSpellcaster(c.monster, c.combatant)
			if !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
				t.Errorf("NewMonsterSpellcaster() = %v, %v, want the zero caster and ErrInvalidInput [%s]", got.IsZero(), err, monsterSpellcastingSpec+c.requirement)
			}
			forParts, err := combat.MonsterSpellcasterForParts(c.monster, c.combatant)
			if c.parts == (err != nil) || (c.parts && forParts.IsZero()) {
				t.Errorf("MonsterSpellcasterForParts() error = %v, want accepted %v", err, c.parts)
			}
			if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("MonsterSpellcasterForParts() error = %v, want ErrInvalidInput", err)
			}
			if _, has := forParts.Spellcasting(); has {
				t.Errorf("a caster for parts holds a trait")
			}
		})
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-033 - a caster compares by value through Equal: the combatant, the monster, the six Modifiers and the trait each count.
// Path: per case.
func TestMonsterSpellcasterEqual(t *testing.T) {
	t.Parallel()
	m := mscMonster(t, nil)
	base := luMust(combat.NewMonsterSpellcaster(m, mscID()))
	if !base.Equal(luMust(combat.NewMonsterSpellcaster(m, mscID()))) {
		t.Errorf("two casters of one monster are not equal")
	}
	scored := luMonster(t, func(f *content.MonsterFields) {
		mscScores(t, f, map[ability.Ability]int{ability.Intelligence: 16, ability.Wisdom: 12, ability.Charisma: 22})
		f.Traits = []content.StatBlockPart{mscTraitPart("Spellcasting", mscTraitFields())}
	})
	retrait := mscMonster(t, func(f *content.MonsterSpellcastingFields) { f.SaveDC = 15 })
	for _, c := range []struct {
		name, requirement, path string
		other                   combat.MonsterSpellcaster
	}{
		{"another combatant", "FR-033", "unhappy (boundary)", luMust(combat.NewMonsterSpellcaster(m, combat.MustCombatantID("other")))},
		{"another modifier", "FR-033", "unhappy (boundary)", luMust(combat.NewMonsterSpellcaster(scored, mscID()))},
		{"another trait", "FR-033", "unhappy (boundary)", luMust(combat.NewMonsterSpellcaster(retrait, mscID()))},
		{"no trait", "FR-033", "unhappy (boundary)", luMust(combat.MonsterSpellcasterForParts(m, mscID()))},
	} {
		if base.Equal(c.other) || c.other.Equal(base) {
			t.Errorf("%s: casters compare equal [%s]", c.name, c.requirement)
		}
	}
	if (combat.MonsterSpellcaster{}).IsZero() != true {
		t.Errorf("the zero caster is not zero")
	}
}
