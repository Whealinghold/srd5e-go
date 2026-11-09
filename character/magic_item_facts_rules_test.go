package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func factsOf(t *testing.T, f content.MagicItemFactsFields) content.MagicItemFacts {
	t.Helper()
	return must(content.NewMagicItemFacts(mid("x"), f))
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-124 and table "Fact resolution").
// Path: per case.
func TestMagicItemFactResolution(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Rods", "Staffs", "Wands", "Potions", pp. 204-206`
	cases := []struct {
		name, requirement, path string
		category                content.MagicItemCategory
		fields                  content.MagicItemFactsFields
		check                   func(character.ResolvedMagicItemFacts) bool
	}{
		{"default holds", p, "happy", content.MagicItemWand, content.MagicItemFactsFields{}, func(r character.ResolvedMagicItemFacts) bool { return r.ArcaneFocus }},
		{"override turns it off", p, "unhappy (rule interaction)", content.MagicItemWand, content.MagicItemFactsFields{ArcaneFocus: content.OverriddenBoolFact(false)}, func(r character.ResolvedMagicItemFacts) bool { return !r.ArcaneFocus }},
		{"override turns it on", p, "unhappy (rule interaction)", content.MagicItemWeapon, content.MagicItemFactsFields{Consumable: content.OverriddenBoolFact(true)}, func(r character.ResolvedMagicItemFacts) bool { return r.Consumable }},
		{"description governs activation", p, "unhappy (rule interaction)", content.MagicItemPotion, content.MagicItemFactsFields{ActivationTime: content.OverriddenActivationTime(content.ActivationDescribedElsewhere)}, func(r character.ResolvedMagicItemFacts) bool {
			return r.ActivationTime == content.ActivationDescribedElsewhere
		}},
		{"other facts unaffected", p, "unhappy (rule interaction)", content.MagicItemStaff, content.MagicItemFactsFields{NonmagicalQuarterstaff: content.OverriddenBoolFact(false)}, func(r character.ResolvedMagicItemFacts) bool { return !r.NonmagicalQuarterstaff && r.ArcaneFocus }},
		{"potion is consumable by default", p, "happy", content.MagicItemPotion, content.MagicItemFactsFields{}, func(r character.ResolvedMagicItemFacts) bool { return r.Consumable }},
		{"must be worn by default for a ring", p, "happy", content.MagicItemRing, content.MagicItemFactsFields{MustBeWorn: content.OverriddenBoolFact(false)}, func(r character.ResolvedMagicItemFacts) bool { return !r.MustBeWorn }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.MagicItemFactResolution{}.Of(c.category, factsOf(t, c.fields))
			if !c.check(got) {
				t.Errorf("Of = %+v [%s]", got, c.requirement)
			}
		})
	}
	r := character.MagicItemFactResolution{}
	if r.EffectiveBool(true, content.StandardBoolFact()) != true || r.EffectiveBool(true, content.OverriddenBoolFact(false)) != false ||
		r.EffectiveActivationTime(content.ActivationAction, content.StandardActivationTime()) != content.ActivationAction ||
		r.EffectiveActivationTime(content.ActivationAction, content.OverriddenActivationTime(content.ActivationBonusAction)) != content.ActivationBonusAction {
		t.Error("Effective")
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-073 - an overridden resilience wins over the content default.
// Path: unhappy (rule interaction).
func TestMagicItemResilienceRulesEffective(t *testing.T) {
	t.Parallel()
	rules := character.MagicItemResilienceRules{}
	def := content.MagicItemResilienceFacts{}.DefaultFor(content.MagicItemRing, content.RarityRare)
	if got := rules.Effective(content.MagicItemRing, content.RarityRare, content.StandardResilience()); got != def {
		t.Errorf("default = %v, want %v", got, def)
	}
	for _, o := range []content.MagicItemResilience{content.ResilienceResistantToAllDamage, content.ResilienceNonmagicalDurability, content.ResilienceImpervious} {
		if got := rules.Effective(content.MagicItemRing, content.RarityRare, content.OverriddenResilience(o)); got != o {
			t.Errorf("override %v = %v (SRD 5.2.1, \"Magic Item Resilience\", p. 206)", o, got)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-111, FR-121 and table "Rarity resolution").
// Path: per case.
func TestResolvedRarityOf(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Magic Item Rarity", p. 205`
	variants := must(content.NewRarityByVariant([]content.RarityOption{
		must(content.NewRarityOption(content.RarityUncommon, "+1")), must(content.NewRarityOption(content.RarityRare, "+2")), must(content.NewRarityOption(content.RarityVeryRare, "+3"))}))
	cases := []struct {
		name, requirement, path string
		spec                    content.RaritySpec
		choice                  character.RarityChoice
		want                    character.RarityResolution
	}{
		{"single", p, "happy", must(content.NewSingleRarity(content.RarityRare)), character.NoRarityChoice(), character.ResolvedRarity{Rarity: content.RarityRare}},
		{"single ignores the choice", p, "happy", must(content.NewSingleRarity(content.RarityRare)), character.ChosenRarity(content.RarityCommon), character.ResolvedRarity{Rarity: content.RarityRare}},
		{"varies, chosen", p, "happy", content.RarityVaries{}, character.ChosenRarity(content.RarityVeryRare), character.ResolvedRarity{Rarity: content.RarityVeryRare}},
		{"varies, no choice", p, "unhappy (missing content)", content.RarityVaries{}, character.NoRarityChoice(), character.RarityNotResolved{Reason: character.RarityChoiceRequired}},
		{"varies, a variant", p, "unhappy (missing content)", content.RarityVaries{}, character.ChosenVariant("+1"), character.RarityNotResolved{Reason: character.RarityChoiceRequired}},
		{"variant, listed", p, "happy", variants, character.ChosenVariant("+2"), character.ResolvedRarity{Rarity: content.RarityRare}},
		{"variant, not listed", p, "unhappy (invalid input)", variants, character.ChosenVariant("+4"), character.RarityNotResolved{Reason: character.RarityVariantNotListed}},
		{"variant, rarity given", p, "unhappy (rule interaction)", variants, character.ChosenRarity(content.RarityRare), character.RarityNotResolved{Reason: character.RarityChoiceRequired}},
		{"variant, no choice", p, "unhappy (rule interaction)", variants, character.NoRarityChoice(), character.RarityNotResolved{Reason: character.RarityChoiceRequired}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.ResolvedRarity{}.Of(c.spec, c.choice)
			if err != nil || got != c.want {
				t.Errorf("Of = %#v, %v, want %#v [%s]", got, err, c.want, c.requirement)
			}
		})
	}
	if _, err := (character.ResolvedRarity{}).Of(nil, character.NoRarityChoice()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil spec: %v", err)
	}
	if character.RarityChoiceRequired.String() != "choice_required" || character.RarityVariantNotListed.String() != "variant_not_listed" || character.RarityNotResolvedReason(0).String() != "unknown" {
		t.Error("String")
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-007).
// Path: per case.
func TestBaseItemSelectionChoose(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Armor", "Weapons", pp. 204-205; "Ammunition, +1, +2, or +3", p. 209`
	plate := armorOf(t, "plate_x", catH, noneRule(t, 18), 15, true, 65, gp(1500))
	sword := weaponOf("sword_x", content.WeaponMelee, gp(15))
	arrows := must(content.NewAmmunitionVariant("arrows_x", "Arrows", noSource, "ammo_family", price(gp(1)), lbs(1), "", "", must(content.NewAmmunitionData(20, "Quiver"))))
	rope := toolOf("rope_x")
	specific := content.BaseItemSpec(must(content.NewSpecificBaseItems([]content.ItemID{"plate_x", "sword_x"})))
	cases := []struct {
		name, requirement, path string
		category                content.MagicItemCategory
		spec                    content.BaseItemSpec
		choice                  content.EquipmentEntry
		allowed                 bool
	}{
		{"specific armor", p, "happy", content.MagicItemArmor, specific, plate, true},
		{"specific, outside the types", p, "unhappy (invalid input)", content.MagicItemWeapon, specific, weaponOf("axe_x", content.WeaponMelee, gp(5)), false},
		{"unspecified armor, an Armor", p, "happy", content.MagicItemArmor, content.UnspecifiedBaseItem{}, plate, true},
		{"unspecified armor, a Weapon", p, "unhappy (invalid input)", content.MagicItemArmor, content.UnspecifiedBaseItem{}, sword, false},
		{"unspecified weapon, a Weapon", p, "happy", content.MagicItemWeapon, content.UnspecifiedBaseItem{}, sword, true},
		{"unspecified weapon, ammunition", p, "happy", content.MagicItemWeapon, content.UnspecifiedBaseItem{}, arrows, true},
		{"unspecified weapon, an Armor", p, "unhappy (invalid input)", content.MagicItemWeapon, content.UnspecifiedBaseItem{}, plate, false},
		{"unspecified weapon, other gear", p, "unhappy (invalid input)", content.MagicItemWeapon, content.UnspecifiedBaseItem{}, rope, false},
		{"unspecified armor, ammunition", p, "unhappy (invalid input)", content.MagicItemArmor, content.UnspecifiedBaseItem{}, arrows, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.BaseItemSelection{}.Choose(c.category, c.spec, c.choice)
			if err != nil {
				t.Fatal(err)
			}
			_, allowed := got.(character.MagicItemBaseItemAllowed)
			if allowed != c.allowed {
				t.Errorf("Choose = %#v, want allowed %v [%s]", got, c.allowed, c.requirement)
			}
		})
	}
	for name, run := range map[string]func() error{
		"nil choice": func() error {
			_, e := character.BaseItemSelection{}.Choose(content.MagicItemArmor, specific, nil)
			return e
		},
		"nil spec": func() error {
			_, e := character.BaseItemSelection{}.Choose(content.MagicItemArmor, nil, plate)
			return e
		},
		"ring category": func() error {
			_, e := character.BaseItemSelection{}.Choose(content.MagicItemRing, specific, plate)
			return e
		},
		"no base item": func() error {
			_, e := character.BaseItemSelection{}.Choose(content.MagicItemArmor, content.NoBaseItem{}, plate)
			return e
		},
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
