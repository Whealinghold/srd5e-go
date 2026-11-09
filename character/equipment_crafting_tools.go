package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ToolsFor returns the id of every tool whose Craft list names item (SRD 5.2.1,
// "Tools", Craft, pp. 93-94; "Crafting Nonmagical Items", p. 103): by id, by
// the family a focus or ammunition variant belongs to, or by a set (melee or
// ranged weapons, medium or heavy armor) that includes it and does not list it
// in its exceptions. A scroll variant is not matched through its family and a
// tool variant matches nothing.
//
// The ids are alternatives: any one tool of the list suffices. The result is
// ordered by id, holds each id once whatever the input order, is a fresh slice
// and is empty when no tool names the item (the GM assigns the tools).
func (Crafting) ToolsFor(item content.EquipmentEntry, tools []content.Tool) []content.ItemID {
	if item == nil {
		return nil
	}
	if _, variant := item.(content.ToolVariant); variant {
		return nil
	}
	names := []content.ItemID{item.ID()}
	switch v := item.(type) {
	case content.FocusVariant:
		names = append(names, v.VariantOf())
	case content.AmmunitionVariant:
		names = append(names, v.VariantOf())
	}
	var out []content.ItemID
	for _, t := range tools {
		if craftListNames(craftList(t), item, names) {
			out = append(out, t.ID())
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func craftList(t content.Tool) []content.CraftReference {
	switch v := t.(type) {
	case content.StandaloneTool:
		return v.Craft()
	case content.ToolFamily:
		return v.Craft()
	}
	return nil // a ToolVariant has no Craft list
}

func craftListNames(list []content.CraftReference, item content.EquipmentEntry, names []content.ItemID) bool {
	for _, ref := range list {
		switch r := ref.(type) {
		case content.CraftItem:
			if slices.Contains(names, r.ID()) {
				return true
			}
		case content.CraftSet:
			if setIncludes(r.Set(), item) && !slices.Contains(r.Except(), item.ID()) {
				return true
			}
		}
	}
	return false
}

// setIncludes reports whether item belongs to the named craft set: a weapon of
// the set's type or an armor of the set's category. Any other entry is outside
// every set.
func setIncludes(set content.CraftSetName, item content.EquipmentEntry) bool {
	switch v := item.(type) {
	case content.Weapon:
		return set == content.CraftMeleeWeapons && v.Type() == content.WeaponMelee ||
			set == content.CraftRangedWeapons && v.Type() == content.WeaponRanged
	case content.Armor:
		return set == content.CraftMediumArmor && v.Category() == content.ArmorMedium ||
			set == content.CraftHeavyArmor && v.Category() == content.ArmorHeavy
	}
	return false
}
