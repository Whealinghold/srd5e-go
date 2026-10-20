package content

import "slices"

// cloneOrNil copies s, mapping an empty slice to nil so that an entry built
// from an empty list equals one built from nil or decoded from "[]".
func cloneOrNil[T any](s []T) []T {
	if len(s) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return nil
	}
	return slices.Clone(s)
}

// Equal reports whether o and p hold the same items in the same order and the
// same gold.
func (o EquipmentOption) Equal(p EquipmentOption) bool {
	return o.gold == p.gold && slices.Equal(o.items, p.items)
}

// Equal reports whether o and p hold equal options A and B.
func (o EquipmentOptions) Equal(p EquipmentOptions) bool {
	return o.a.Equal(p.a) && o.b.Equal(p.b)
}

// Equal reports whether o and p have the same name, text and spell ids in the
// same order.
func (o TextOption) Equal(p TextOption) bool {
	return o.name == p.name && o.text == p.text && slices.Equal(o.spellIDs, p.spellIDs)
}

// choiceOptionEqual reports whether a and b are the same variant with equal
// members. Only TextOption holds a slice, so the other variants use ==.
func choiceOptionEqual(a, b ChoiceOption) bool {
	if ta, ok := a.(TextOption); ok {
		tb, ok := b.(TextOption)
		return ok && ta.Equal(tb)
	}
	if _, ok := b.(TextOption); ok {
		return false
	}
	return a == b
}

// Equal reports whether c and d name the same trait with equal options in the
// same order.
func (c SpeciesChoice) Equal(d SpeciesChoice) bool {
	return c.traitName == d.traitName && slices.EqualFunc(c.options, d.options, choiceOptionEqual)
}

// Equal reports whether s and o are the same species: every member equal, the
// lists in the same order.
func (s Species) Equal(o Species) bool {
	return s.id == o.id && s.name == o.name && s.source == o.source && s.creatureType == o.creatureType &&
		s.sizeText == o.sizeText && s.speed == o.speed && slices.Equal(s.sizes, o.sizes) &&
		slices.Equal(s.traits, o.traits) && slices.EqualFunc(s.choices, o.choices, SpeciesChoice.Equal)
}

// prerequisiteEqual reports whether a and b are the same variant with equal
// members. Only AbilityScorePrerequisite holds a slice.
func prerequisiteEqual(a, b Prerequisite) bool {
	if pa, ok := a.(AbilityScorePrerequisite); ok {
		pb, ok := b.(AbilityScorePrerequisite)
		return ok && pa.Equal(pb)
	}
	if _, ok := b.(AbilityScorePrerequisite); ok {
		return false
	}
	return a == b
}

// Equal reports whether f and o are the same feat: every member equal, the
// lists in the same order.
func (f Feat) Equal(o Feat) bool {
	return f.id == o.id && f.name == o.name && f.source == o.source && f.category == o.category &&
		f.repeatable == o.repeatable && f.repeatableText == o.repeatableText &&
		slices.Equal(f.benefits, o.benefits) && slices.EqualFunc(f.prerequisites, o.prerequisites, prerequisiteEqual)
}

// Equal reports whether b and o are the same background: every member equal,
// the lists in the same order.
func (b Background) Equal(o Background) bool {
	return b.id == o.id && b.name == o.name && b.source == o.source && b.feat == o.feat &&
		b.toolProficiency == o.toolProficiency && slices.Equal(b.abilityScores, o.abilityScores) &&
		slices.Equal(b.skillProficiencies, o.skillProficiencies) && b.equipmentOptions.Equal(o.equipmentOptions)
}
