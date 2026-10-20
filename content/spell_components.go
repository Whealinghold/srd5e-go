package content

import "slices"

// MinMaterialCostGP is the least figure of a material component cost in Gold
// Pieces (content-spells.md FR-101, FR-111).
const MinMaterialCostGP = 1

// ValidMaterialCostGP reports whether n is a cost figure a material component
// may list, at least [MinMaterialCostGP].
func ValidMaterialCostGP(n int) bool { return n >= MinMaterialCostGP }

// MaterialComponent is the Material (M) component of a spell: its verbatim
// description, the Gold Piece costs it states and whether the spell consumes
// the materials (SRD 5.2.1, "Components", pp. 105-106). The zero value is
// invalid; build it with [NewMaterialComponent]. It holds a slice, so it is not
// comparable with ==; use [MaterialComponent.Equal].
type MaterialComponent struct {
	description string
	costsGP     []int
	consumes    bool
}

// NewMaterialComponent returns the component described by description (kept
// verbatim), copying costsGP, the figures "N+ GP" in printed order.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText or BlockInvalidCharacter for a description that breaks the
// text rule (schema FR-028) and BlockOutOfRange for a cost below
// [MinMaterialCostGP]. An empty costsGP is stored as none. It never panics.
func NewMaterialComponent(description string, costsGP []int, consumesMaterial bool) (MaterialComponent, error) {
	if err := checkText("MaterialComponent", description); err != nil {
		return MaterialComponent{}, err
	}
	if slices.ContainsFunc(costsGP, func(n int) bool { return !ValidMaterialCostGP(n) }) {
		return MaterialComponent{}, blockError("MaterialComponent", BlockOutOfRange)
	}
	return MaterialComponent{description: description, costsGP: cloneOrNil(costsGP), consumes: consumesMaterial}, nil
}

// Description returns the verbatim description.
func (m MaterialComponent) Description() string { return m.description }

// CostsGP returns the stated costs in Gold Pieces in printed order, as a new
// slice, or nil when the description states none.
func (m MaterialComponent) CostsGP() []int { return slices.Clone(m.costsGP) }

// ConsumesMaterial reports whether the description states that the spell
// consumes the materials.
func (m MaterialComponent) ConsumesMaterial() bool { return m.consumes }

// Equal reports whether m and o have the same description, costs and
// consumption.
func (m MaterialComponent) Equal(o MaterialComponent) bool {
	return m.description == o.description && m.consumes == o.consumes && slices.Equal(m.costsGP, o.costsGP)
}

// SpellComponents are the Verbal, Somatic and Material components of a spell
// (SRD 5.2.1, "Components", p. 105). The zero value has no component and is
// valid. It holds a slice, so it is not comparable with ==; use
// [SpellComponents.Equal].
type SpellComponents struct {
	verbal, somatic bool
	material        MaterialComponent
	hasMaterial     bool
}

// NewSpellComponents returns the components with Verbal and Somatic as given
// and no Material component.
func NewSpellComponents(verbal, somatic bool) SpellComponents {
	return SpellComponents{verbal: verbal, somatic: somatic}
}

// WithMaterial returns a copy of c that also has the Material component m.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with the reason
// of the component's own check for an invalid one (the zero component
// included), and the zero SpellComponents. It never panics.
func (c SpellComponents) WithMaterial(m MaterialComponent) (SpellComponents, error) {
	if _, err := NewMaterialComponent(m.description, m.costsGP, m.consumes); err != nil {
		return SpellComponents{}, err
	}
	c.material, c.hasMaterial = MaterialComponent{description: m.description, costsGP: cloneOrNil(m.costsGP), consumes: m.consumes}, true
	return c, nil
}

// Verbal reports whether the spell has a Verbal component.
func (c SpellComponents) Verbal() bool { return c.verbal }

// Somatic reports whether the spell has a Somatic component.
func (c SpellComponents) Somatic() bool { return c.somatic }

// Material returns the Material component, and false when the spell has none.
func (c SpellComponents) Material() (MaterialComponent, bool) {
	return MaterialComponent{description: c.material.description, costsGP: cloneOrNil(c.material.costsGP), consumes: c.material.consumes}, c.hasMaterial
}

// Equal reports whether c and o hold the same components.
func (c SpellComponents) Equal(o SpellComponents) bool {
	return c.verbal == o.verbal && c.somatic == o.somatic && c.hasMaterial == o.hasMaterial && c.material.Equal(o.material)
}
