package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// DivineOrderGrants are the additions a Divine Order makes to the Core Cleric Traits:
// the Protector gains proficiency with Martial weapons and training with Heavy armor
// (SRD 5.2.1, "Level 1: Divine Order", p. 37). The zero value grants nothing.
type DivineOrderGrants struct{ MartialWeapons, HeavyArmor bool }

// Grants returns the proficiencies the order adds; none for the Thaumaturge and for no order.
func (o DivineOrder) Grants() DivineOrderGrants {
	return DivineOrderGrants{MartialWeapons: o == DivineOrderProtector, HeavyArmor: o == DivineOrderProtector}
}

// ThaumaturgeCheckBonus returns the bonus a Thaumaturge adds to an Intelligence (Arcana)
// or Intelligence (Religion) check: the Wisdom modifier, at least +1 (SRD 5.2.1, "Level 1:
// Divine Order", p. 37). The second result is false, and the bonus +0, for another order,
// no order or another skill. It never panics.
func ThaumaturgeCheckBonus(order DivineOrder, wisdom ability.Modifier, skill content.Skill) (modifier.Modifier, bool) {
	if order != DivineOrderThaumaturge || skill != content.SkillArcana && skill != content.SkillReligion {
		return modifier.Modifier{}, false
	}
	return modifier.New(max(wisdom.Value(), 1)), true
}

// divineOrderOf reads the requested order names: none is missing, more than one is both,
// a name other than "protector" or "thaumaturge" is unknown. The reason is 0 for a valid name.
func divineOrderOf(names []string) (DivineOrder, ClericReason) {
	switch {
	case len(names) == 0:
		return 0, ClericOrderMissing
	case len(names) > 1:
		return 0, ClericBothOrders
	case names[0] == DivineOrderProtector.String():
		return DivineOrderProtector, 0
	case names[0] == DivineOrderThaumaturge.String():
		return DivineOrderThaumaturge, 0
	}
	return 0, ClericOrderUnknown
}

// divineOrderProblem applies the rules of ChooseDivineOrder after already_chosen: the order
// names, then for a Thaumaturge the extra cantrip, which must be a level 0 spell with Cleric
// in its class list that is not already known. A Protector ignores any extra cantrip.
func divineOrderProblem(names []string, extra *content.Spell, known []content.SpellID) (DivineOrder, ClericReason) {
	order, reason := divineOrderOf(names)
	if reason != 0 || order != DivineOrderThaumaturge {
		return order, reason
	}
	switch {
	case extra == nil:
		return order, ClericExtraCantripMissing
	case extra.Level() != 0 || !slices.Contains(extra.Classes(), ClericClassID):
		return order, ClericExtraCantripNotClericCantrip
	case slices.Contains(known, extra.ID()):
		return order, ClericExtraCantripAlreadyKnown
	}
	return order, 0
}

// ChooseDivineOrderResult is the closed set of outcomes of ChooseDivineOrder. Variants:
// ChooseDivineOrderChosen, ChooseDivineOrderRejected, ChooseDivineOrderNotACleric.
type ChooseDivineOrderResult interface{ isChooseDivineOrderResult() }

// ChooseDivineOrderChosen is the outcome of a valid choice. Only ChooseDivineOrder builds one.
type ChooseDivineOrderChosen struct {
	character Character
	choices   ClericChoices
	events    []Event
}

// Character returns the new Character, holding the choices under ClericChoicesKey.
func (r ChooseDivineOrderChosen) Character() Character { return r.character }

// Choices returns the new choices.
func (r ChooseDivineOrderChosen) Choices() ClericChoices { return r.choices }

// Grants returns the proficiencies the order adds to the Core Cleric Traits.
func (r ChooseDivineOrderChosen) Grants() DivineOrderGrants { return r.choices.order.Grants() }

// ExtraCantrip returns the Thaumaturge's extra cantrip, or "" for a Protector.
func (r ChooseDivineOrderChosen) ExtraCantrip() content.SpellID { return r.choices.extraCantrip }

// ExtraCantrips returns the allowance the host passes as the Cleric's extraCantrips: 1 or 0.
func (r ChooseDivineOrderChosen) ExtraCantrips() int { return r.choices.ExtraCantrips() }

// Events returns a new slice holding the ClericOptionChosen event.
func (r ChooseDivineOrderChosen) Events() []Event { return slices.Clone(r.events) }

// ChooseDivineOrderRejected is the outcome of a refused choice: nothing changed and there are
// no events. Construct it with a keyed literal.
type ChooseDivineOrderRejected struct{ Reason ClericReason }

// ChooseDivineOrderNotACleric is the outcome for a character with no Cleric levels.
type ChooseDivineOrderNotACleric struct{}

func (ChooseDivineOrderChosen) isChooseDivineOrderResult()     {}
func (ChooseDivineOrderRejected) isChooseDivineOrderResult()   {}
func (ChooseDivineOrderNotACleric) isChooseDivineOrderResult() {}

// ChooseDivineOrder chooses the Divine Order of a Cleric (SRD 5.2.1, "Level 1: Divine Order",
// p. 37), at the first Cleric level of a single-class or a multiclass character. names are the
// requested orders ("protector", "thaumaturge"); extra is the Thaumaturge's extra cantrip, which
// is checked against known (the cantrips already known). The first reason that holds is
// reported: already_chosen, order_missing, both_orders, order_unknown, extra_cantrip_missing,
// extra_cantrip_not_cleric_cantrip, extra_cantrip_already_known.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or another type of
// state under ClericChoicesKey. A refused choice is a ChooseDivineOrderRejected result.
func ChooseDivineOrder(c Character, names []string, extra *content.Spell, known []content.SpellID) (ChooseDivineOrderResult, error) {
	if c.IsZero() {
		return nil, invalidInput("ChooseDivineOrder: zero Character")
	}
	if _, ok := clericLevelOf(c); !ok {
		return ChooseDivineOrderNotACleric{}, nil
	}
	held, err := readClericChoices(c)
	if err != nil {
		return nil, err
	}
	if held.order != 0 {
		return ChooseDivineOrderRejected{Reason: ClericAlreadyChosen}, nil
	}
	order, reason := divineOrderProblem(names, extra, known)
	if reason != 0 {
		return ChooseDivineOrderRejected{Reason: reason}, nil
	}
	held.order = order
	if order == DivineOrderThaumaturge {
		held.extraCantrip = extra.ID()
	}
	next, err := WithFeatureState(c, ClericChoicesKey(), held)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		return nil, err
	}
	return ChooseDivineOrderChosen{character: next, choices: held,
		events: []Event{ClericOptionChosen{id: c.ID(), feature: "divine_order", name: order.String()}}}, nil
}
