package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// ResolvedMagicItemFacts are an item's category-default facts after its
// description's exceptions are applied.
type ResolvedMagicItemFacts struct {
	Consumable, ArcaneFocus, NonmagicalQuarterstaff, MustBeWorn bool
	ActivationTime                                              content.ActivationTime
}

// MagicItemFactResolution applies an item's overrides to the category defaults.
// The zero value is ready to use.
type MagicItemFactResolution struct{}

// EffectiveBool returns the override's value when it is set, otherwise standard.
func (MagicItemFactResolution) EffectiveBool(standard bool, fact content.BoolFactOverride) bool {
	if v, set := fact.Value(); set {
		return v
	}
	return standard
}

// EffectiveActivationTime returns the override's value when it is set, otherwise standard.
func (MagicItemFactResolution) EffectiveActivationTime(standard content.ActivationTime, fact content.ActivationTimeOverride) content.ActivationTime {
	if v, set := fact.Value(); set {
		return v
	}
	return standard
}

// EffectiveResilience returns the override's value when it is set, otherwise standard.
func (MagicItemFactResolution) EffectiveResilience(standard content.MagicItemResilience, fact content.ResilienceOverride) content.MagicItemResilience {
	if v, set := fact.Value(); set {
		return v
	}
	return standard
}

// Of applies facts' overrides to the defaults of category, one fact at a time,
// so an item departs from the default for that fact only (SRD 5.2.1, "Armor",
// "Potions", "Rings", "Rods", p. 204).
func (r MagicItemFactResolution) Of(category content.MagicItemCategory, facts content.MagicItemFacts) ResolvedMagicItemFacts {
	d := content.MagicItemCategoryFacts{}.Of(category)
	return ResolvedMagicItemFacts{
		Consumable:             r.EffectiveBool(d.Consumable(), facts.Consumable()),
		ArcaneFocus:            r.EffectiveBool(d.ArcaneFocus(), facts.ArcaneFocus()),
		NonmagicalQuarterstaff: r.EffectiveBool(d.NonmagicalQuarterstaff(), facts.NonmagicalQuarterstaff()),
		MustBeWorn:             r.EffectiveBool(d.MustBeWorn(), facts.MustBeWorn()),
		ActivationTime:         r.EffectiveActivationTime(d.ActivationTime(), facts.ActivationTime()),
	}
}

// MagicItemResilienceRules decides an item's resilience. The zero value is ready to use.
type MagicItemResilienceRules struct{}

// Effective returns the override when it is set and the content default for the
// category and rarity otherwise (SRD 5.2.1, "Magic Item Resilience", p. 206).
func (MagicItemResilienceRules) Effective(category content.MagicItemCategory, rarity content.ItemRarity, override content.ResilienceOverride) content.MagicItemResilience {
	return MagicItemFactResolution{}.EffectiveResilience(content.MagicItemResilienceFacts{}.DefaultFor(category, rarity), override)
}

// RarityChoice is the rarity the host picks for an item that has none of its
// own. The zero value is NoRarityChoice. Construct with NoRarityChoice,
// ChosenRarity or ChosenVariant.
type RarityChoice struct {
	kind    int
	rarity  content.ItemRarity
	variant string
}

// NoRarityChoice is no choice.
func NoRarityChoice() RarityChoice { return RarityChoice{} }

// ChosenRarity is a rarity picked directly.
func ChosenRarity(r content.ItemRarity) RarityChoice { return RarityChoice{kind: 1, rarity: r} }

// ChosenVariant is a variant picked by its text.
func ChosenVariant(text string) RarityChoice { return RarityChoice{kind: 2, variant: text} }

// RarityNotResolvedReason says why a rarity could not be resolved. The zero
// value is not a reason.
type RarityNotResolvedReason int

// The reasons.
const (
	RarityChoiceRequired RarityNotResolvedReason = iota + 1
	RarityVariantNotListed
)

// String returns the snake_case name of the reason, or "unknown".
func (r RarityNotResolvedReason) String() string {
	switch r {
	case RarityChoiceRequired:
		return "choice_required"
	case RarityVariantNotListed:
		return "variant_not_listed"
	}
	return "unknown"
}

// RarityResolution is the closed set of outcomes of ResolvedRarity.Of.
// Variants: ResolvedRarity, RarityNotResolved.
type RarityResolution interface{ isRarityResolution() }

// ResolvedRarity is the one rarity an item has for value and crafting. Of is
// called on its zero value.
type ResolvedRarity struct{ Rarity content.ItemRarity }

// RarityNotResolved: no single rarity could be chosen.
type RarityNotResolved struct{ Reason RarityNotResolvedReason }

func (ResolvedRarity) isRarityResolution()    {}
func (RarityNotResolved) isRarityResolution() {}

// Of resolves spec: SingleRarity gives its rarity; RarityVaries the ChosenRarity
// and otherwise RarityChoiceRequired; RarityByVariant the option whose variant
// text equals the ChosenVariant, RarityVariantNotListed for other text and
// RarityChoiceRequired for another choice. The package never picks a variant. It
// returns an error matching srderr.ErrInvalidInput for a nil or foreign spec.
func (ResolvedRarity) Of(spec content.RaritySpec, choice RarityChoice) (RarityResolution, error) {
	switch s := spec.(type) {
	case content.SingleRarity:
		return ResolvedRarity{Rarity: s.Rarity()}, nil
	case content.RarityVaries:
		if choice.kind == 1 {
			return ResolvedRarity{Rarity: choice.rarity}, nil
		}
		return RarityNotResolved{Reason: RarityChoiceRequired}, nil
	case content.RarityByVariant:
		if choice.kind != 2 {
			return RarityNotResolved{Reason: RarityChoiceRequired}, nil
		}
		for _, o := range s.Options() {
			if o.Variant() == choice.variant {
				return ResolvedRarity{Rarity: o.Rarity()}, nil
			}
		}
		return RarityNotResolved{Reason: RarityVariantNotListed}, nil
	}
	return nil, invalidInput("resolve rarity: nil or foreign rarity spec")
}

// BaseItemSelectionResult is the closed set of outcomes of BaseItemSelection.Choose.
// Variants: MagicItemBaseItemAllowed, MagicItemBaseItemNotAllowed.
type BaseItemSelectionResult interface{ isBaseItemSelectionResult() }

// MagicItemBaseItemAllowed: the chosen entry is a valid base item.
type MagicItemBaseItemAllowed struct{ Entry content.EquipmentEntry }

// MagicItemBaseItemNotAllowed: the chosen entry does not fit the spec or category.
type MagicItemBaseItemNotAllowed struct{}

func (MagicItemBaseItemAllowed) isBaseItemSelectionResult()    {}
func (MagicItemBaseItemNotAllowed) isBaseItemSelectionResult() {}

// BaseItemSelection checks the base item a host chose for a magic Armor or
// Weapon. The zero value is ready to use.
type BaseItemSelection struct{}

// Choose accepts choice when the spec admits it: for SpecificBaseItems one of
// the listed ids; for UnspecifiedBaseItem an Armor for category armor, a Weapon
// or an AmmunitionVariant for category weapon (SRD 5.2.1, "Armor", "Weapons",
// pp. 204-205). It returns an error matching srderr.ErrInvalidInput for a nil
// choice or spec, or a category other than armor or weapon.
func (BaseItemSelection) Choose(category content.MagicItemCategory, spec content.BaseItemSpec, choice content.EquipmentEntry) (BaseItemSelectionResult, error) {
	if choice == nil || spec == nil || category != content.MagicItemArmor && category != content.MagicItemWeapon {
		return nil, invalidInput("choose base item: nil choice or spec, or category not armor or weapon")
	}
	var fits bool
	switch s := spec.(type) {
	case content.SpecificBaseItems:
		for _, id := range s.Items() {
			fits = fits || id == choice.ID()
		}
	case content.UnspecifiedBaseItem:
		switch choice.(type) {
		case content.Armor:
			fits = category == content.MagicItemArmor
		case content.Weapon, content.AmmunitionVariant:
			fits = category == content.MagicItemWeapon
		}
	default:
		return nil, invalidInput("choose base item: foreign spec")
	}
	if !fits {
		return MagicItemBaseItemNotAllowed{}, nil
	}
	return MagicItemBaseItemAllowed{Entry: choice}, nil
}
