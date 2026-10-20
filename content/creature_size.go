package content

// CreatureSize is one of the six creature sizes, in ascending order (SRD
// 5.2.1, "Monsters", "Size", p. 254), so sizes compare with <. The zero
// CreatureSize is invalid. Comparable and safe for concurrent use.
type CreatureSize int

// The creature sizes.
const (
	SizeTiny CreatureSize = iota + 1
	SizeSmall
	SizeMedium
	SizeLarge
	SizeHuge
	SizeGargantuan
)

func creatureSizeNames() []string {
	return []string{"Tiny", "Small", "Medium", "Large", "Huge", "Gargantuan"}
}

// Valid reports whether z is one of the six sizes.
func (z CreatureSize) Valid() bool { return enumValid(creatureSizeNames(), int(z)) }

// String returns the printed name of z, or "unknown" for an undefined value.
func (z CreatureSize) String() string { return enumString(creatureSizeNames(), int(z)) }

// ParseCreatureSize returns the size spelled exactly s, such as "Large".
// Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseCreatureSize(s string) (CreatureSize, error) {
	v, err := parseEnum("CreatureSize", creatureSizeNames(), s)
	return CreatureSize(v), err
}

// CreatureSpace is the row of the Creature Size and Space table for one size
// (SRD 5.2.1, "Creature Size", p. 14). It is a closed set with exactly two
// variants, obtainable only from [CreatureSize.Space]: [SharedSquareSpace]
// for Tiny and [WholeSquareSpace] for every other size. Consumers switch over
// it exhaustively. Variants are immutable, comparable and safe for concurrent
// use.
type CreatureSpace interface {
	// SideFeet returns the side of the space in feet.
	SideFeet() float64
	isCreatureSpace()
}

// SharedSquareSpace is the space of a Tiny creature: several share one
// 5-foot square. It has no exported constructor.
type SharedSquareSpace struct{ creaturesPerSquare int }

// SideFeet returns the side of the space in feet (2.5, exact in binary).
func (SharedSquareSpace) SideFeet() float64 { return 2.5 }

// CreaturesPerSquare returns how many creatures share one square.
func (s SharedSquareSpace) CreaturesPerSquare() int { return s.creaturesPerSquare }

func (SharedSquareSpace) isCreatureSpace() {}

// WholeSquareSpace is the space of a Small or larger creature: a whole number
// of squares along each side. It has no exported constructor.
type WholeSquareSpace struct{ squaresPerSide int }

// SideFeet returns the side of the space in feet, 5 per square.
func (w WholeSquareSpace) SideFeet() float64 { return 5 * float64(w.squaresPerSide) }

// SquaresPerSide returns how many squares make one side.
func (w WholeSquareSpace) SquaresPerSide() int { return w.squaresPerSide }

// Squares returns the squares the space covers: SquaresPerSide squared.
func (w WholeSquareSpace) Squares() int { return w.squaresPerSide * w.squaresPerSide }

func (WholeSquareSpace) isCreatureSpace() {}

// Space returns the Creature Size and Space row for z: a [SharedSquareSpace]
// for Tiny, a [WholeSquareSpace] for the others. It returns nil when z is
// invalid.
func (z CreatureSize) Space() CreatureSpace {
	switch z {
	case SizeTiny:
		return SharedSquareSpace{creaturesPerSquare: 4}
	case SizeSmall, SizeMedium:
		return WholeSquareSpace{squaresPerSide: 1}
	case SizeLarge:
		return WholeSquareSpace{squaresPerSide: 2}
	case SizeHuge:
		return WholeSquareSpace{squaresPerSide: 3}
	case SizeGargantuan:
		return WholeSquareSpace{squaresPerSide: 4}
	}
	return nil
}
