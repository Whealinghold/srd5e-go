package character

import "github.com/Whealinghold/srd5e-go/content"

// LightSourceRules is the stateless service that turns a light source and a
// distance into the light level that source alone gives (SRD 5.2.1, "Light",
// p. 11). The zero value is ready to use. Safe for concurrent use.
type LightSourceRules struct{}

// LevelAt returns the light level emission alone gives at space. A distance equal
// to a radius is inside it. Within brightFeet it is content.LightBright, out to
// brightFeet plus dimFeet content.LightDim, and beyond that content.LightDarkness. For
// a Cone emission a space not within the Cone is content.LightDarkness at every
// distance; a radius emission ignores that report. The zero LightEmission gives
// content.LightDarkness. The result is not the space's final level: combine it with
// Strongest.
func (LightSourceRules) LevelAt(emission LightEmission, space LightSpace) content.LightLevel {
	switch {
	case emission.IsZero(), emission.shape == LightShapeCone && !space.withinCone:
		return content.LightDarkness
	case emission.brightFeet > 0 && space.distanceFeet <= emission.brightFeet:
		return content.LightBright
	case space.distanceFeet <= emission.brightFeet+emission.dimFeet:
		return content.LightDim
	}
	return content.LightDarkness
}

// Strongest returns the brightest of levels (Bright Light, then Dim Light, then
// Darkness), and content.LightDarkness for none. Invalid levels are ignored. The SRD
// gives no combination rule; this is a package decision.
func (LightSourceRules) Strongest(levels []content.LightLevel) content.LightLevel {
	best := content.LightDarkness
	for _, l := range levels {
		if l.Valid() && l < best {
			best = l
		}
	}
	return best
}

// HoodedLanternHood is the state of a Hooded Lantern's hood, a value the host
// holds. The zero value is invalid. Comparable.
type HoodedLanternHood int

// The hood states.
const (
	// HoodRaised: the lantern lights a 30-foot radius.
	HoodRaised HoodedLanternHood = iota + 1
	// HoodLowered: the lantern sheds Dim Light in a 5-foot radius.
	HoodLowered
)

// SrdLightSources holds the emissions of the gear in the "Adventuring Gear" table
// that gives a radius (SRD 5.2.1, pp. 96-100). Items whose text gives no radius have
// none. The zero value is ready to use. Safe for concurrent use.
type SrdLightSources struct{}

// Candle returns Bright Light in a 5-foot radius and Dim Light for 5 more feet.
func (SrdLightSources) Candle() LightEmission { return MustLightEmission(LightShapeRadius, 5, 5) }

// Lamp returns Bright Light in a 15-foot radius and Dim Light for 30 more feet.
func (SrdLightSources) Lamp() LightEmission { return MustLightEmission(LightShapeRadius, 15, 30) }

// BullseyeLantern returns Bright Light in a 60-foot Cone and Dim Light for 60 more
// feet.
func (SrdLightSources) BullseyeLantern() LightEmission {
	return MustLightEmission(LightShapeCone, 60, 60)
}

// Torch returns Bright Light in a 20-foot radius and Dim Light for 20 more feet.
func (SrdLightSources) Torch() LightEmission { return MustLightEmission(LightShapeRadius, 20, 20) }

// HoodedLantern returns the emission of a Hooded Lantern with hood raised (30-foot
// radius, Dim Light 30 more feet) or lowered (Dim Light in a 5-foot radius, no
// Bright Light). An invalid hood gives the zero LightEmission.
func (SrdLightSources) HoodedLantern(hood HoodedLanternHood) LightEmission {
	switch hood {
	case HoodRaised:
		return MustLightEmission(LightShapeRadius, 30, 30)
	case HoodLowered:
		return MustLightEmission(LightShapeRadius, 0, 5)
	}
	return LightEmission{}
}

// All returns the six emissions in a new slice: Candle, Lamp, BullseyeLantern,
// HoodedLantern raised, HoodedLantern lowered and Torch.
func (s SrdLightSources) All() []LightEmission {
	return []LightEmission{s.Candle(), s.Lamp(), s.BullseyeLantern(), s.HoodedLantern(HoodRaised), s.HoodedLantern(HoodLowered), s.Torch()}
}
