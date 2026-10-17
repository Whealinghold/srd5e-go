package modifier

// RollCircumstance is a circumstance that grants Advantage or imposes
// Disadvantage on a d20 Test. It carries no data (no cause, no source).
//
// The variants are exactly Advantage and Disadvantage; the interface cannot be
// implemented outside this package, so a type switch over those two cases is
// exhaustive. Use the values, not
// pointers: a *Advantage or *Disadvantage also satisfies the interface but is
// ignored by Resolve, like nil.
type RollCircumstance interface {
	rollCircumstance()
}

// Advantage is the circumstance that grants Advantage.
type Advantage struct{}

// Disadvantage is the circumstance that imposes Disadvantage.
type Disadvantage struct{}

func (Advantage) rollCircumstance()    {}
func (Disadvantage) rollCircumstance() {}

// String returns "Advantage".
func (Advantage) String() string { return "Advantage" }

// String returns "Disadvantage".
func (Disadvantage) String() string { return "Disadvantage" }
