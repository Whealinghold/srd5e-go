package content

// AreaShape is the closed set of the six areas of effect of SRD 5.2.1 ("Area
// of Effect", p. 177): [ConeArea], [CubeArea], [CylinderArea], [EmanationArea],
// [LineArea] and [SphereArea], each with its dimensions in whole feet. These
// are the only definitions of the six shapes in the package. Every variant is
// an immutable, comparable value and safe for concurrent use. Consumers switch
// over it exhaustively.
type AreaShape interface {
	// IncludesOrigin reports the static default of the shape: whether its
	// point of origin is in the area unless its creator decides otherwise.
	IncludesOrigin() bool
	isAreaShape()
}

func areaDimension(block string, feet int) error {
	if feet < 1 {
		return blockError(block, BlockOutOfRange)
	}
	return nil
}

// ConeArea is a Cone (SRD 5.2.1, "Cone", p. 179). The zero value is invalid;
// build it with [NewConeArea].
type ConeArea struct{ length int }

// NewConeArea returns a cone of lengthFeet feet.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a length below 1. It never panics.
func NewConeArea(lengthFeet int) (ConeArea, error) {
	if err := areaDimension("ConeArea", lengthFeet); err != nil {
		return ConeArea{}, err
	}
	return ConeArea{length: lengthFeet}, nil
}

// LengthFeet returns the length of the cone in feet.
func (a ConeArea) LengthFeet() int { return a.length }

// IncludesOrigin reports false: a Cone's point of origin is not in it.
func (ConeArea) IncludesOrigin() bool { return false }
func (ConeArea) isAreaShape()         {}

// CubeArea is a Cube (SRD 5.2.1, "Cube", p. 179). The zero value is invalid;
// build it with [NewCubeArea].
type CubeArea struct{ side int }

// NewCubeArea returns a cube of sideFeet feet.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a side below 1. It never panics.
func NewCubeArea(sideFeet int) (CubeArea, error) {
	if err := areaDimension("CubeArea", sideFeet); err != nil {
		return CubeArea{}, err
	}
	return CubeArea{side: sideFeet}, nil
}

// SideFeet returns the side of the cube in feet.
func (a CubeArea) SideFeet() int { return a.side }

// IncludesOrigin reports false: a Cube's point of origin is not in it.
func (CubeArea) IncludesOrigin() bool { return false }
func (CubeArea) isAreaShape()         {}

// CylinderArea is a Cylinder (SRD 5.2.1, "Cylinder", p. 180). The zero value
// is invalid; build it with [NewCylinderArea].
type CylinderArea struct{ radius, height int }

// NewCylinderArea returns a cylinder of radiusFeet and heightFeet feet.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a radius or a height below 1. It never panics.
func NewCylinderArea(radiusFeet, heightFeet int) (CylinderArea, error) {
	if err := areaDimension("CylinderArea", radiusFeet); err != nil {
		return CylinderArea{}, err
	}
	if err := areaDimension("CylinderArea", heightFeet); err != nil {
		return CylinderArea{}, err
	}
	return CylinderArea{radius: radiusFeet, height: heightFeet}, nil
}

// RadiusFeet returns the radius of the cylinder in feet.
func (a CylinderArea) RadiusFeet() int { return a.radius }

// HeightFeet returns the height of the cylinder in feet.
func (a CylinderArea) HeightFeet() int { return a.height }

// IncludesOrigin reports true: a Cylinder's point of origin is in it.
func (CylinderArea) IncludesOrigin() bool { return true }
func (CylinderArea) isAreaShape()         {}

// EmanationArea is an Emanation (SRD 5.2.1, "Emanation", p. 181). The zero
// value is invalid; build it with [NewEmanationArea].
type EmanationArea struct{ distance int }

// NewEmanationArea returns an emanation reaching distanceFeet feet.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a distance below 1. It never panics.
func NewEmanationArea(distanceFeet int) (EmanationArea, error) {
	if err := areaDimension("EmanationArea", distanceFeet); err != nil {
		return EmanationArea{}, err
	}
	return EmanationArea{distance: distanceFeet}, nil
}

// DistanceFeet returns how far the emanation reaches in feet.
func (a EmanationArea) DistanceFeet() int { return a.distance }

// IncludesOrigin reports false: an Emanation's point of origin is not in it.
func (EmanationArea) IncludesOrigin() bool { return false }
func (EmanationArea) isAreaShape()         {}

// LineArea is a Line (SRD 5.2.1, "Line", p. 184). The zero value is invalid;
// build it with [NewLineArea].
type LineArea struct{ length, width int }

// NewLineArea returns a line of lengthFeet and widthFeet feet.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a length or a width below 1. It never panics.
func NewLineArea(lengthFeet, widthFeet int) (LineArea, error) {
	if err := areaDimension("LineArea", lengthFeet); err != nil {
		return LineArea{}, err
	}
	if err := areaDimension("LineArea", widthFeet); err != nil {
		return LineArea{}, err
	}
	return LineArea{length: lengthFeet, width: widthFeet}, nil
}

// LengthFeet returns the length of the line in feet.
func (a LineArea) LengthFeet() int { return a.length }

// WidthFeet returns the width of the line in feet.
func (a LineArea) WidthFeet() int { return a.width }

// IncludesOrigin reports false: a Line's point of origin is not in it.
func (LineArea) IncludesOrigin() bool { return false }
func (LineArea) isAreaShape()         {}

// SphereArea is a Sphere (SRD 5.2.1, "Sphere", p. 188). The zero value is
// invalid; build it with [NewSphereArea].
type SphereArea struct{ radius int }

// NewSphereArea returns a sphere of radiusFeet feet.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a radius below 1. It never panics.
func NewSphereArea(radiusFeet int) (SphereArea, error) {
	if err := areaDimension("SphereArea", radiusFeet); err != nil {
		return SphereArea{}, err
	}
	return SphereArea{radius: radiusFeet}, nil
}

// RadiusFeet returns the radius of the sphere in feet.
func (a SphereArea) RadiusFeet() int { return a.radius }

// IncludesOrigin reports true: a Sphere's point of origin is in it.
func (SphereArea) IncludesOrigin() bool { return true }
func (SphereArea) isAreaShape()         {}

// checkAreaShape reports why s is not a shape its constructor would accept:
// BlockMissing for nil.
func checkAreaShape(s AreaShape) error {
	var err error
	switch v := s.(type) {
	case ConeArea:
		_, err = NewConeArea(v.length)
	case CubeArea:
		_, err = NewCubeArea(v.side)
	case CylinderArea:
		_, err = NewCylinderArea(v.radius, v.height)
	case EmanationArea:
		_, err = NewEmanationArea(v.distance)
	case LineArea:
		_, err = NewLineArea(v.length, v.width)
	case SphereArea:
		_, err = NewSphereArea(v.radius)
	default:
		err = blockError("AreaShape", BlockMissing)
	}
	return err
}
