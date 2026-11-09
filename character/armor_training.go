package character

import "github.com/Whealinghold/srd5e-go/content"

// ArmorTraining is a set of armor categories a character is trained in (SRD
// 5.2.1, "Armor Training", p. 177; "Armor", p. 92). It is an immutable value,
// comparable with ==. The zero value is the empty set. Safe for concurrent use.
type ArmorTraining struct{ set uint8 }

// NewArmorTraining returns the set of categories; order and repeats are
// irrelevant. It returns an error matching srderr.ErrInvalidInput for an
// invalid category. It never panics.
func NewArmorTraining(categories ...content.ArmorCategory) (ArmorTraining, error) {
	var s uint8
	for _, c := range categories {
		if !c.Valid() {
			return ArmorTraining{}, invalidInput("invalid armor category")
		}
		s |= 1 << c
	}
	return ArmorTraining{set: s}, nil
}

// Has reports whether the set holds c.
func (t ArmorTraining) Has(c content.ArmorCategory) bool { return c.Valid() && t.set&(1<<c) != 0 }

// Union returns the set of the categories in t or o.
func (t ArmorTraining) Union(o ArmorTraining) ArmorTraining { return ArmorTraining{set: t.set | o.set} }

// Categories returns a new slice of the categories held, in the order Light,
// Medium, Heavy, Shield.
func (t ArmorTraining) Categories() []content.ArmorCategory {
	var out []content.ArmorCategory
	for _, c := range []content.ArmorCategory{content.ArmorLight, content.ArmorMedium, content.ArmorHeavy, content.ArmorShield} {
		if t.Has(c) {
			out = append(out, c)
		}
	}
	return out
}

// classArmorTraining converts the armor training of a class definition, a set of
// content.ArmorTrainingCategory, to an ArmorTraining by member name (light,
// medium, heavy, shield). It is the one place the two mirrored enums meet.
func classArmorTraining(def content.ClassDefinition) ArmorTraining {
	var t ArmorTraining
	for _, c := range def.ArmorTraining() {
		switch c {
		case content.ArmorTrainingLight:
			t.set |= 1 << content.ArmorLight
		case content.ArmorTrainingMedium:
			t.set |= 1 << content.ArmorMedium
		case content.ArmorTrainingHeavy:
			t.set |= 1 << content.ArmorHeavy
		case content.ArmorTrainingShield:
			t.set |= 1 << content.ArmorShield
		}
	}
	return t
}
