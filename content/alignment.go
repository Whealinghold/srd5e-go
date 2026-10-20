package content

// Alignment is one of the ten values a stat block prints: the nine alignments
// then Unaligned (SRD 5.2.1, "The Nine Alignments", p. 21; "Unaligned
// Creatures", p. 22; "Alignment", p. 254). The wire spelling is the full
// printed name, never the abbreviation. It carries no rules. The zero
// Alignment is invalid. Comparable and safe for concurrent use.
type Alignment int

// The alignments.
const (
	AlignmentLawfulGood Alignment = iota + 1
	AlignmentNeutralGood
	AlignmentChaoticGood
	AlignmentLawfulNeutral
	AlignmentNeutral
	AlignmentChaoticNeutral
	AlignmentLawfulEvil
	AlignmentNeutralEvil
	AlignmentChaoticEvil
	AlignmentUnaligned
)

func alignmentNames() []string {
	return []string{
		"Lawful Good", "Neutral Good", "Chaotic Good",
		"Lawful Neutral", "Neutral", "Chaotic Neutral",
		"Lawful Evil", "Neutral Evil", "Chaotic Evil", "Unaligned",
	}
}

// Valid reports whether a is one of the ten alignments.
func (a Alignment) Valid() bool { return enumValid(alignmentNames(), int(a)) }

// String returns the printed name of a, or "unknown" for an undefined value.
func (a Alignment) String() string { return enumString(alignmentNames(), int(a)) }

// ParseAlignment returns the alignment spelled exactly s, such as "Lawful
// Good". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string, including "LG" and "Any Alignment". It never panics.
func ParseAlignment(s string) (Alignment, error) {
	v, err := parseEnum("Alignment", alignmentNames(), s)
	return Alignment(v), err
}
