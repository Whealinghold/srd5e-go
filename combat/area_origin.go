package combat

import "github.com/Whealinghold/srd5e-go/content"

// ConeWidthAt returns the width in feet of cone at distance feet from its point of
// origin along its length: equal to the distance (SRD 5.2.1, "Cone", p. 179: "a
// cone's width at a given point along its length is equal to that point's
// distance from the point of origin"). The distance 0 gives width 0 and the cone's
// length gives the length. It returns an error matching srderr.ErrInvalidInput
// for a distance below 0 or above the cone's length, and for the zero (invalid)
// cone. It never panics. This is a function, not a method of content.ConeArea,
// because package combat cannot add a method to a type of content.
func ConeWidthAt(cone content.ConeArea, distance int) (int, error) {
	if cone.LengthFeet() < 1 {
		return 0, invalidInput("cone has no length")
	}
	if distance < 0 || distance > cone.LengthFeet() {
		return 0, invalidInput("distance is outside the cone")
	}
	return distance, nil
}

// OverridableAreaShape is the closed set of the four shapes whose point of origin
// is not included by default, and so the only ones whose creator may decide to
// include it: OverridableCone, OverridableCube, OverridableEmanation and
// OverridableLine. A Cylinder or Sphere always includes its origin (SRD 5.2.1,
// pp. 180, 188), so no variant wraps one and such an override cannot be built.
// Shape returns the wrapped shape.
type OverridableAreaShape interface {
	Shape() content.AreaShape
	isOverridableAreaShape()
}

// OverridableCone wraps a Cone. Construct with a keyed literal; the zero value
// holds the invalid zero cone and CreatorIncludesOrigin rejects it.
type OverridableCone struct{ Area content.ConeArea }

// OverridableCube wraps a Cube. Construct with a keyed literal; the zero value is invalid.
type OverridableCube struct{ Area content.CubeArea }

// OverridableEmanation wraps an Emanation. Construct with a keyed literal; the zero value is invalid.
type OverridableEmanation struct{ Area content.EmanationArea }

// OverridableLine wraps a Line. Construct with a keyed literal; the zero value is invalid.
type OverridableLine struct{ Area content.LineArea }

// Shape returns the wrapped cone.
func (o OverridableCone) Shape() content.AreaShape { return o.Area }

// Shape returns the wrapped cube.
func (o OverridableCube) Shape() content.AreaShape { return o.Area }

// Shape returns the wrapped emanation.
func (o OverridableEmanation) Shape() content.AreaShape { return o.Area }

// Shape returns the wrapped line.
func (o OverridableLine) Shape() content.AreaShape { return o.Area }

func (OverridableCone) isOverridableAreaShape()      {}
func (OverridableCube) isOverridableAreaShape()      {}
func (OverridableEmanation) isOverridableAreaShape() {}
func (OverridableLine) isOverridableAreaShape()      {}

// validAreaShape reports whether s is one of the six shapes with every dimension
// at least 1 feet (the zero value of each shape fails).
func validAreaShape(s content.AreaShape) bool {
	switch v := s.(type) {
	case content.ConeArea:
		return v.LengthFeet() >= 1
	case content.CubeArea:
		return v.SideFeet() >= 1
	case content.CylinderArea:
		return v.RadiusFeet() >= 1 && v.HeightFeet() >= 1
	case content.EmanationArea:
		return v.DistanceFeet() >= 1
	case content.LineArea:
		return v.LengthFeet() >= 1 && v.WidthFeet() >= 1
	case content.SphereArea:
		return v.RadiusFeet() >= 1
	}
	return false
}

// OriginInclusionDecision says whether an area's point of origin is in the area:
// the shape's default, or the creator's decision to include it (SRD 5.2.1, "Cone",
// p. 179: "unless its creator decides otherwise"). Build it with ShapeDefaultOrigin
// or CreatorIncludesOrigin; the zero value is invalid. Comparable and safe for
// concurrent use.
type OriginInclusionDecision struct {
	shape     content.AreaShape
	included  bool
	byCreator bool
}

// ShapeDefaultOrigin returns the decision that follows the shape's own default:
// OriginIncluded equals shape.IncludesOrigin (false for Cone, Cube, Emanation and
// Line; true for Cylinder and Sphere). It returns an error matching
// srderr.ErrInvalidInput for a nil or invalid shape. It never panics.
func ShapeDefaultOrigin(shape content.AreaShape) (OriginInclusionDecision, error) {
	if !validAreaShape(shape) {
		return OriginInclusionDecision{}, invalidInput("origin decision needs a valid area shape")
	}
	return OriginInclusionDecision{shape: shape, included: shape.IncludesOrigin()}, nil
}

// CreatorIncludesOrigin returns the decision of a creator who includes the origin
// of a shape whose origin is not included by default. OriginIncluded is true. It
// returns an error matching srderr.ErrInvalidInput for a nil variant or an invalid
// (zero) shape. It never panics.
func CreatorIncludesOrigin(shape OverridableAreaShape) (OriginInclusionDecision, error) {
	if shape == nil || !validAreaShape(shape.Shape()) {
		return OriginInclusionDecision{}, invalidInput("origin decision needs a valid overridable area shape")
	}
	return OriginInclusionDecision{shape: shape.Shape(), included: true, byCreator: true}, nil
}

// OriginIncluded reports whether the point of origin is in the area.
func (d OriginInclusionDecision) OriginIncluded() bool { return d.included }

// ByCreator reports whether the creator decided to include the origin.
func (d OriginInclusionDecision) ByCreator() bool { return d.byCreator }

// Shape returns the shape the decision is about, nil for the zero value.
func (d OriginInclusionDecision) Shape() content.AreaShape { return d.shape }

// IsZero reports whether d is the invalid zero value.
func (d OriginInclusionDecision) IsZero() bool { return d.shape == nil }

// EmanationEffectTiming is how long the effect of an Emanation lasts, which
// decides whether the Emanation moves. The zero value is invalid.
type EmanationEffectTiming int

// The timings of an Emanation's effect.
const (
	// EmanationInstantaneous: the effect is instantaneous.
	EmanationInstantaneous EmanationEffectTiming = iota + 1
	// EmanationStationary: the effect is stationary.
	EmanationStationary
	// EmanationOngoing: neither instantaneous nor stationary.
	EmanationOngoing
)

// String returns "instantaneous", "stationary", "ongoing" or "unknown".
func (t EmanationEffectTiming) String() string {
	return enumName([]string{"instantaneous", "stationary", "ongoing"}, int(t))
}

// MovesWithOrigin reports whether an Emanation moves with its origin creature or
// object: it does unless the effect is instantaneous or stationary (SRD 5.2.1,
// "Emanation", p. 181). It is false for an undefined timing.
func (t EmanationEffectTiming) MovesWithOrigin() bool { return t == EmanationOngoing }

// OriginPlacement is where the creator places an area's point of origin. The zero
// value is invalid.
type OriginPlacement int

// The placements of a point of origin.
const (
	// SeenPoint: a point the creator sees.
	SeenPoint OriginPlacement = iota + 1
	// UnseenPointNoObstruction: an unseen point with nothing between it and the creator.
	UnseenPointNoObstruction
	// UnseenPointBehindObstruction: an unseen point with an obstruction between it and the creator.
	UnseenPointBehindObstruction
)

// AreaOriginResolution is where the origin of an area ends up. The zero value is
// invalid.
type AreaOriginResolution int

// The resolutions of a placement.
const (
	// ChosenPoint: the origin is the point the creator chose.
	ChosenPoint AreaOriginResolution = iota + 1
	// NearSideOfObstruction: the origin is on the near side of the obstruction.
	NearSideOfObstruction
)

// String returns "chosen_point", "near_side_of_obstruction" or "unknown".
func (r AreaOriginResolution) String() string {
	return enumName([]string{"chosen_point", "near_side_of_obstruction"}, int(r))
}

// AreaOrigin is the stateless service for the origin of an area. The zero value is
// ready to use. Safe for concurrent use.
type AreaOrigin struct{}

// Resolve returns NearSideOfObstruction when the creator places the origin at an
// unseen point and an obstruction lies between the creator and that point, and
// ChosenPoint for the other two placements (SRD 5.2.1, "Area of Effect", p. 177).
// An undefined placement gives the zero (invalid) resolution. It never panics.
func (AreaOrigin) Resolve(p OriginPlacement) AreaOriginResolution {
	switch p {
	case SeenPoint, UnseenPointNoObstruction:
		return ChosenPoint
	case UnseenPointBehindObstruction:
		return NearSideOfObstruction
	}
	return 0
}
