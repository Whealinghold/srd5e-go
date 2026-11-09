package character

// LightShape is the shape of the area a light source lights: a radius around it
// or a Cone (SRD 5.2.1, "Lantern, Bullseye", p. 98; "Cone [Area of Effect]",
// p. 179). The Cone's geometry is the host's. The zero LightShape is invalid.
// Comparable and safe for concurrent use.
type LightShape int

// The light shapes.
const (
	// LightShapeRadius lights every space within the distance of the source.
	LightShapeRadius LightShape = iota + 1
	// LightShapeCone lights only the spaces the host reports as within the Cone.
	LightShapeCone
)

// Valid reports whether s is a radius or a Cone.
func (s LightShape) Valid() bool { return s == LightShapeRadius || s == LightShapeCone }

// String returns "radius", "cone" or "unknown".
func (s LightShape) String() string {
	switch s {
	case LightShapeRadius:
		return "radius"
	case LightShapeCone:
		return "cone"
	}
	return "unknown"
}

// LightEmission is the light a source sheds: Bright Light out to brightFeet and Dim
// Light for an additional dimFeet (SRD 5.2.1, "Torch", p. 100). The zero
// LightEmission is invalid; see IsZero. Comparable and safe for concurrent use.
type LightEmission struct {
	shape               LightShape
	brightFeet, dimFeet int
}

// NewLightEmission returns an emission. It returns a *ExplorationError (matching
// srderr.ErrInvalidInput) for an invalid shape, a negative distance, or both
// distances 0 (a source that sheds no light). It never panics.
func NewLightEmission(shape LightShape, brightFeet, dimFeet int) (LightEmission, error) {
	switch {
	case !shape.Valid():
		return LightEmission{}, &ExplorationError{Input: ExplorationLightShape}
	case brightFeet < 0 || dimFeet < 0:
		return LightEmission{}, &ExplorationError{Input: ExplorationLightRadius}
	case brightFeet == 0 && dimFeet == 0:
		return LightEmission{}, &ExplorationError{Input: ExplorationNoLight}
	}
	return LightEmission{shape: shape, brightFeet: brightFeet, dimFeet: dimFeet}, nil
}

// MustLightEmission is NewLightEmission for constants known to be valid. It panics
// with the error NewLightEmission returns, and only then.
func MustLightEmission(shape LightShape, brightFeet, dimFeet int) LightEmission {
	e, err := NewLightEmission(shape, brightFeet, dimFeet)
	if err != nil {
		panic(err)
	}
	return e
}

// Shape returns the shape.
func (e LightEmission) Shape() LightShape { return e.shape }

// BrightFeet returns the distance of Bright Light.
func (e LightEmission) BrightFeet() int { return e.brightFeet }

// DimFeet returns the Dim Light distance beyond BrightFeet.
func (e LightEmission) DimFeet() int { return e.dimFeet }

// IsZero reports whether e is the invalid zero LightEmission.
func (e LightEmission) IsZero() bool { return e == LightEmission{} }

// LightSpace is a space relative to a light source: its distance from the source
// and, for a Cone, whether the host reports it within the Cone. The engine has no
// map. The zero LightSpace is valid: the source's own space, outside any Cone.
// Comparable and safe for concurrent use.
type LightSpace struct {
	distanceFeet int
	withinCone   bool
}

// NewLightSpace returns a space distanceFeet from the source. It returns a
// *ExplorationError (matching srderr.ErrInvalidInput) for a negative distance;
// 0 is the source's own space. It never panics.
func NewLightSpace(distanceFeet int, withinCone bool) (LightSpace, error) {
	if distanceFeet < 0 {
		return LightSpace{}, &ExplorationError{Input: ExplorationLightDistance}
	}
	return LightSpace{distanceFeet: distanceFeet, withinCone: withinCone}, nil
}

// DistanceFeet returns the distance from the source.
func (s LightSpace) DistanceFeet() int { return s.distanceFeet }

// WithinCone returns whether the host reports the space within the Cone.
func (s LightSpace) WithinCone() bool { return s.withinCone }
