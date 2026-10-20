package content

// FeatCategory is the category of a feat, in SRD order (SRD 5.2.1, "Parts of
// a Feat: Category", p. 87). The wire spelling is the printed name, spaces
// included. The zero FeatCategory is invalid. Comparable and safe for
// concurrent use.
type FeatCategory int

// The feat categories.
const (
	FeatOrigin FeatCategory = iota + 1
	FeatGeneral
	FeatFightingStyle
	FeatEpicBoon
)

func featCategoryNames() []string {
	return []string{"Origin", "General", "Fighting Style", "Epic Boon"}
}

// Valid reports whether c is one of the four categories.
func (c FeatCategory) Valid() bool { return enumValid(featCategoryNames(), int(c)) }

// String returns the printed name of c, or "unknown" for an undefined value.
func (c FeatCategory) String() string { return enumString(featCategoryNames(), int(c)) }

// ParseFeatCategory returns the category spelled exactly s, such as
// "Fighting Style". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string, including "origin". It never panics.
func ParseFeatCategory(s string) (FeatCategory, error) {
	v, err := parseEnum("FeatCategory", featCategoryNames(), s)
	return FeatCategory(v), err
}
