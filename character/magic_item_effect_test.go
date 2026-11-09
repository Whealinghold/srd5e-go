package character_test

import (
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func equipped(t *testing.T, id, instance string, choice character.EnhancementChoice, ctx character.ItemEffectContext) character.EquippedItem {
	t.Helper()
	return character.EquippedItem{Instance: inst(instance), Effects: effectsOf(t, id, choice).Effects(), Context: ctx}
}

func ctxOf(access character.MagicalPropertiesAccessLevel, usage character.ItemUsage) character.ItemEffectContext {
	return character.ItemEffectContext{Access: access, Usage: usage}
}

// Requirement: doc/specs/magic-item-effects.md FR-009 to FR-012 (SRD 5.2.1, "Attunement", p. 102; "Bracers of Defense", p. 214; "Ring of Protection",
// p. 238) - every row of table "Bonus combination": effects count only at fullMagical and under their condition, bonuses from different items add
// without a cap and each is listed with its item copy, in input order.
// Path: per case.
func TestActiveEffectsOfBonusCombination(t *testing.T) {
	t.Parallel()
	full, nonmagical, none := character.AccessFullMagical, character.AccessNonmagicalBenefitsOnly, character.AccessNoMagic
	worn, held := character.UsageWorn, character.UsageHeld
	plus := func(r content.ItemRarity) character.EnhancementChoice { return character.EnhancementChoice{Rarity: r} }
	cases := []struct {
		name      string
		items     []character.EquippedItem
		ac, saves []int
		instances []string
	}{
		{"ring and cloak", []character.EquippedItem{equipped(t, "ring_of_protection", "r-1", plus(0), ctxOf(full, worn)), equipped(t, "cloak_of_protection", "c-1", plus(0), ctxOf(full, worn))}, []int{1, 1}, []int{1, 1}, []string{"r-1", "c-1"}},
		{"cloak not attuned", []character.EquippedItem{equipped(t, "cloak_of_protection", "c-1", plus(0), ctxOf(nonmagical, worn))}, []int{}, []int{}, nil},
		{"cloak with no magic", []character.EquippedItem{equipped(t, "cloak_of_protection", "c-1", plus(0), ctxOf(none, worn))}, []int{}, []int{}, nil},
		{"bracers with armor", []character.EquippedItem{equipped(t, "bracers_of_defense", "b-1", plus(0), character.ItemEffectContext{Access: full, Usage: worn, WearsArmor: true})}, []int{}, []int{}, nil},
		{"bracers with a shield", []character.EquippedItem{equipped(t, "bracers_of_defense", "b-1", plus(0), character.ItemEffectContext{Access: full, Usage: worn, UsesShield: true})}, []int{}, []int{}, nil},
		{"bracers bare", []character.EquippedItem{equipped(t, "bracers_of_defense", "b-1", plus(0), ctxOf(full, worn))}, []int{2}, []int{}, []string{"b-1"}},
		{"+2 armor and +1 shield", []character.EquippedItem{equipped(t, "armor_plus", "a-1", plus(content.RarityVeryRare), ctxOf(full, worn)), equipped(t, "shield_plus", "s-1", plus(content.RarityUncommon), ctxOf(full, held))}, []int{2, 1}, []int{}, []string{"a-1", "s-1"}},
		{"shield not held", []character.EquippedItem{equipped(t, "shield_plus", "s-1", plus(content.RarityUncommon), ctxOf(full, worn))}, []int{}, []int{}, nil},
		{"no items", nil, []int{}, []int{}, nil},
		{"empty slice", []character.EquippedItem{}, []int{}, []int{}, nil},
		{"item without effects", []character.EquippedItem{{Instance: inst("x-1"), Context: ctxOf(full, worn)}}, []int{}, []int{}, nil},
		{"unset context", []character.EquippedItem{equipped(t, "ring_of_protection", "r-1", plus(0), character.ItemEffectContext{})}, []int{}, []int{}, nil},
		{"no cap: five bonuses", []character.EquippedItem{
			equipped(t, "ring_of_protection", "r-1", plus(0), ctxOf(full, worn)), equipped(t, "cloak_of_protection", "c-1", plus(0), ctxOf(full, worn)),
			equipped(t, "elven_chain", "e-1", plus(0), ctxOf(full, worn)), equipped(t, "dwarven_plate", "d-1", plus(0), ctxOf(full, worn)), equipped(t, "scarab_of_protection", "sc-1", plus(0), ctxOf(full, character.UsageCarried))},
			[]int{1, 1, 1, 2, 1}, []int{1, 1}, []string{"r-1", "c-1", "e-1", "d-1", "sc-1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := must(character.ActiveEffectsOf(c.items))
			if !eqInts(bonusValues(got.ArmorClass()), c.ac) || !eqInts(bonusValues(got.SavingThrows()), c.saves) {
				t.Fatalf("ac %v saves %v, want %v %v", bonusValues(got.ArmorClass()), bonusValues(got.SavingThrows()), c.ac, c.saves)
			}
			for i, b := range got.ArmorClass() {
				if i < len(c.instances) && string(b.InstanceID()) != c.instances[i] {
					t.Errorf("bonus %d from %s, want %s", i, b.InstanceID(), c.instances[i])
				}
			}
			if got.IgnoresHalfCoverOnSpellAttacks() {
				t.Error("half cover flag set")
			}
		})
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-010 - each ItemEffectCondition holds for exactly its usage: worn, held, on person (any use), and
// worn without armor or Shield.
// Path: per case.
func TestItemEffectConditions(t *testing.T) {
	t.Parallel()
	full := character.AccessFullMagical
	type use struct {
		usage         character.ItemUsage
		armor, shield bool
	}
	uses := []use{{character.UsageWorn, false, false}, {character.UsageHeld, false, false}, {character.UsageCarried, false, false}, {0, false, false},
		{character.UsageWorn, true, false}, {character.UsageWorn, false, true}, {character.UsageCarried + 1, false, false}}
	want := map[character.ItemEffectCondition][]bool{
		character.WhileWorn:                     {true, false, false, false, true, true, false},
		character.WhileHeld:                     {false, true, false, false, false, false, false},
		character.WhileOnPerson:                 {true, true, true, false, true, true, false},
		character.WhileWornWithoutArmorOrShield: {true, false, false, false, false, false, false},
		0:                                       {false, false, false, false, false, false, false},
		character.WhileWornWithoutArmorOrShield + 1: {false, false, false, false, false, false, false},
	}
	for cond, w := range want {
		for i, u := range uses {
			item := character.EquippedItem{Instance: inst("i-1"), Effects: []character.MagicItemEffect{character.NewArmorClassBonus(modifier.New(1), cond)},
				Context: character.ItemEffectContext{Access: full, Usage: u.usage, WearsArmor: u.armor, UsesShield: u.shield}}
			got := len(must(character.ActiveEffectsOf([]character.EquippedItem{item})).ArmorClass()) == 1
			if got != w[i] {
				t.Errorf("condition %d, use %+v = %v, want %v", cond, u, got, w[i])
			}
		}
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-012 - ActiveItemEffects exposes each kind as a copied slice (so the lists cannot be altered), carries
// the Half Cover flag, and ActiveEffectsOf does not change its input and rejects a blank instance.
// Path: unhappy (immutability).
func TestActiveEffectsOfKindsAndCopies(t *testing.T) {
	t.Parallel()
	staff := equipped(t, "staff_of_power", "sp-1", character.EnhancementChoice{}, ctxOf(character.AccessFullMagical, character.UsageHeld))
	wand := equipped(t, "wand_of_the_war_mage_plus", "w-1", character.EnhancementChoice{Rarity: content.RarityUncommon}, ctxOf(character.AccessFullMagical, character.UsageHeld))
	items := []character.EquippedItem{staff, wand}
	got := must(character.ActiveEffectsOf(items))
	if !eqInts(bonusValues(got.SpellAttackRolls()), []int{2, 1}) || !eqInts(bonusValues(got.AttackAndDamage()), []int{2}) || !eqInts(bonusValues(got.SavingThrows()), []int{2}) ||
		!got.IgnoresHalfCoverOnSpellAttacks() {
		t.Errorf("active = %+v", got)
	}
	for _, list := range [][]character.MagicItemBonus{got.ArmorClass(), got.SavingThrows(), got.SpellAttackRolls(), got.AttackAndDamage()} {
		if len(list) > 0 {
			list[0] = character.MagicItemBonus{}
		}
	}
	if !eqInts(bonusValues(got.SpellAttackRolls()), []int{2, 1}) || !eqInts(bonusValues(got.ArmorClass()), []int{2}) || !eqInts(bonusValues(got.SavingThrows()), []int{2}) || !eqInts(bonusValues(got.AttackAndDamage()), []int{2}) {
		t.Error("an exposed slice aliases the value")
	}
	if len(items) != 2 || len(items[0].Effects) != 4 {
		t.Error("the input changed")
	}
	if _, err := character.ActiveEffectsOf([]character.EquippedItem{{Instance: "", Effects: staff.Effects, Context: staff.Context}}); err == nil {
		t.Error("blank instance accepted")
	} else {
		wantInvalid(t, err, "blank instance")
	}
	var zero character.ActiveItemEffects
	if len(zero.ArmorClass()) != 0 || zero.IgnoresHalfCoverOnSpellAttacks() {
		t.Error("zero value is not empty")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-014, FR-018, FR-019, FR-021 - the score effect constructors reject an invalid ability, a score or cap
// outside 1 to 30 and an amount below 1, and the accessors return what was given.
// Path: unhappy (invalid input).
func TestScoreEffectConstructors(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		err  error
	}{
		{"set zero ability", second(character.NewSetAbilityScore(0, 19, character.WhileWorn))},
		{"set score 0", second(character.NewSetAbilityScore(ability.Strength, 0, character.WhileWorn))},
		{"set score 31", second(character.NewSetAbilityScore(ability.Strength, 31, character.WhileWorn))},
		{"raise zero ability", second(character.NewRaiseAbilityScore(0, 2, 20, character.WhileWorn))},
		{"raise amount 0", second(character.NewRaiseAbilityScore(ability.Constitution, 0, 20, character.WhileWorn))},
		{"raise cap 0", second(character.NewRaiseAbilityScore(ability.Constitution, 2, 0, character.WhileWorn))},
		{"raise cap 31", second(character.NewRaiseAbilityScore(ability.Constitution, 2, 31, character.WhileWorn))},
		{"boost amount 0", second(character.NewBoostBestowedScore(0, 30, character.WhileWorn))},
		{"boost cap 0", second(character.NewBoostBestowedScore(4, 0, character.WhileWorn))},
		{"boost cap 31", second(character.NewBoostBestowedScore(4, 31, character.WhileWorn))},
	} {
		wantInvalid(t, c.err, c.name)
	}
	set := must(character.NewSetAbilityScore(ability.Strength, 30, character.WhileWorn))
	raise := must(character.NewRaiseAbilityScore(ability.Constitution, 1, 1, character.WhileHeld))
	boost := must(character.NewBoostBestowedScore(1, 1, character.WhileOnPerson))
	if set.Score().Value() != 30 || set.Condition() != character.WhileWorn || set.Duration() != 0 || set.Lasting(time.Hour).Duration() != time.Hour ||
		raise.Cap() != 1 || raise.Amount() != 1 || raise.Condition() != character.WhileHeld || boost.Cap() != 1 || boost.Amount() != 1 {
		t.Errorf("accessors: %+v %+v %+v", set, raise, boost)
	}
}

func second[T any](_ T, err error) error { return err }
