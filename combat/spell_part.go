package combat

// PrismaticLayer is the number of one layer of a Prismatic Wall, 1 (red) to 7
// (violet) in the order of SRD 5.2.1, "Prismatic Wall", p. 155. Build it with
// NewPrismaticLayer; a value outside 1 to 7 is rejected by every function that
// takes one. Comparable and safe for concurrent use.
type PrismaticLayer int

// NewPrismaticLayer returns the layer n. It returns an error matching
// srderr.ErrInvalidInput for n outside 1 to 7. It never panics.
func NewPrismaticLayer(n int) (PrismaticLayer, error) {
	if n < 1 || n > 7 {
		return 0, invalidInput("prismatic layer must be 1 to 7")
	}
	return PrismaticLayer(n), nil
}

// Valid reports whether l is a layer, 1 to 7.
func (l PrismaticLayer) Valid() bool { return l >= 1 && l <= 7 }

// spellPartKind names which part of a multi-part spell an entry is.
type spellPartKind int

const (
	spellPartWhole spellPartKind = iota
	spellPartPrismaticLayer
	spellPartGuardsAndWardsEffect
)

// SpellPart names which part of a multi-part spell a registry entry is. The
// zero SpellPart is the whole spell or its parent (SRD 5.2.1, "Prismatic Wall",
// p. 155, and "Guards and Wards", p. 138). Comparable and safe for concurrent use.
type SpellPart struct {
	kind  spellPartKind
	layer PrismaticLayer
}

// SpellPartPrismaticLayer returns the part that is one layer of a Prismatic
// Wall. A layer outside 1 to 7 makes DispelRestrictions.ForSpell return an
// error matching srderr.ErrInvalidInput.
func SpellPartPrismaticLayer(l PrismaticLayer) SpellPart {
	return SpellPart{kind: spellPartPrismaticLayer, layer: l}
}

// SpellPartGuardsAndWardsEffect returns the part that is one of the four
// effects of Guards and Wards (SRD 5.2.1, p. 138).
func SpellPartGuardsAndWardsEffect() SpellPart {
	return SpellPart{kind: spellPartGuardsAndWardsEffect}
}

// IsWhole reports whether p is the whole spell or its parent (the zero value).
func (p SpellPart) IsWhole() bool { return p.kind == spellPartWhole }
