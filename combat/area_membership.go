package combat

import "slices"

// ShapeContainment is whether the host reports a location as inside the geometry
// of an area's shape. The engine models no map, so containment is a host fact. The
// zero value is invalid.
type ShapeContainment int

// The two containments.
const (
	// InsideShape: the location is inside the shape.
	InsideShape ShapeContainment = iota + 1
	// OutsideShape: the location is outside the shape.
	OutsideShape
)

// AreaExclusion is why a location is not in an area. The zero value is not a reason.
type AreaExclusion int

// The reasons a location is not in an area.
const (
	// ExcludedOutsideShape: the host reports the location outside the shape.
	ExcludedOutsideShape AreaExclusion = iota + 1
	// ExcludedAllLinesBlocked: every straight line from the origin is blocked by Total Cover.
	ExcludedAllLinesBlocked
	// ExcludedOrigin: the location is the point of origin and the origin is not included.
	ExcludedOrigin
)

// String returns "outside_shape", "all_lines_blocked", "origin_excluded" or "unknown".
func (r AreaExclusion) String() string {
	return enumName([]string{"outside_shape", "all_lines_blocked", "origin_excluded"}, int(r))
}

// AreaLocationFacts is the closed set of what the host reports about a location
// when it asks whether the location is in an area. Variants: OriginLocationFacts,
// OtherLocationFacts.
type AreaLocationFacts interface{ isAreaLocationFacts() }

// OriginLocationFacts says the location is the area's point of origin. Construct it
// with NewOriginLocationFacts; the zero value is invalid.
type OriginLocationFacts struct{ inclusion OriginInclusionDecision }

// NewOriginLocationFacts returns the facts for the origin location under the
// decision. It returns an error matching srderr.ErrInvalidInput for the zero
// decision. It never panics.
func NewOriginLocationFacts(d OriginInclusionDecision) (OriginLocationFacts, error) {
	if d.IsZero() {
		return OriginLocationFacts{}, invalidInput("origin location needs an inclusion decision")
	}
	return OriginLocationFacts{d}, nil
}

// OtherLocationFacts is the host's containment of any other location and the Cover
// each straight line from the origin to it passes through. Construct it with
// NewOtherLocationFacts; the zero value is invalid. It holds a slice, so it is not
// comparable with ==.
type OtherLocationFacts struct {
	containment ShapeContainment
	lines       []Cover
}

// NewOtherLocationFacts returns the facts. lines holds, per straight line, the most
// protective Cover on it (no cover is the zero Cover); it is copied. It returns an
// error matching srderr.ErrInvalidInput for an undefined containment or a line that
// is not a valid Cover, and a LimitExceededError for more than 256 lines (plan P10, checked first). It never panics.
func NewOtherLocationFacts(containment ShapeContainment, lines []Cover) (OtherLocationFacts, error) {
	if err := overCap("area lines", len(lines), maxInputList); err != nil {
		return OtherLocationFacts{}, err
	}
	if containment != InsideShape && containment != OutsideShape {
		return OtherLocationFacts{}, invalidInput("location needs a containment")
	}
	for _, c := range lines {
		if !c.Valid() {
			return OtherLocationFacts{}, invalidInput("line has an invalid Cover")
		}
	}
	return OtherLocationFacts{containment, slices.Clone(lines)}, nil
}

// Containment returns the containment.
func (f OtherLocationFacts) Containment() ShapeContainment { return f.containment }

// Lines returns a copy of the covers of the straight lines.
func (f OtherLocationFacts) Lines() []Cover { return slices.Clone(f.lines) }

func (OriginLocationFacts) isAreaLocationFacts() {}
func (OtherLocationFacts) isAreaLocationFacts()  {}

// AreaMembershipResult is the closed set of answers of AreaMembership.Assess. An
// excluded location is an applied answer, not a refusal. Variants:
// AreaMembershipIncluded, AreaMembershipExcluded, AreaMembershipNoLinesConsidered.
type AreaMembershipResult interface{ isAreaMembershipResult() }

// AreaMembershipIncluded: the location is in the area.
type AreaMembershipIncluded struct{}

// AreaMembershipExcluded: the location is not in the area, for the reason.
type AreaMembershipExcluded struct{ Reason AreaExclusion }

// AreaMembershipNoLinesConsidered: the facts cannot decide the question: an inside
// location with no lines ("no lines considered" is neither open nor blocked), or
// facts that are nil or the invalid zero value. The host's input is malformed.
type AreaMembershipNoLinesConsidered struct{}

func (AreaMembershipIncluded) isAreaMembershipResult()          {}
func (AreaMembershipExcluded) isAreaMembershipResult()          {}
func (AreaMembershipNoLinesConsidered) isAreaMembershipResult() {}

// AreaMembership is the stateless service that decides whether a location is in an
// area from host facts. The zero value is ready to use. Safe for concurrent use.
type AreaMembership struct{}

// Assess decides whether the location is in the area (SRD 5.2.1, "Area of Effect",
// p. 177). The origin is in the area exactly when its decision says so. Any other
// location is checked in this order: outside the shape is Excluded
// (ExcludedOutsideShape) whatever its lines, even none; an inside location with no
// lines is NoLinesConsidered; an inside location with at least one line that is not
// Total Cover is Included (only Total Cover blocks a line, p. 15); an inside
// location whose every line is Total Cover is Excluded (ExcludedAllLinesBlocked).
// It does not change its argument and never panics.
func (AreaMembership) Assess(facts AreaLocationFacts) AreaMembershipResult {
	switch f := facts.(type) {
	case OriginLocationFacts:
		if f.inclusion.IsZero() {
			return AreaMembershipNoLinesConsidered{}
		}
		if f.inclusion.OriginIncluded() {
			return AreaMembershipIncluded{}
		}
		return AreaMembershipExcluded{ExcludedOrigin}
	case OtherLocationFacts:
		switch {
		case f.containment == OutsideShape:
			return AreaMembershipExcluded{ExcludedOutsideShape}
		case f.containment != InsideShape || len(f.lines) == 0:
			return AreaMembershipNoLinesConsidered{}
		case slices.ContainsFunc(f.lines, func(c Cover) bool { return c != TotalCover }):
			return AreaMembershipIncluded{}
		}
		return AreaMembershipExcluded{ExcludedAllLinesBlocked}
	}
	return AreaMembershipNoLinesConsidered{}
}
