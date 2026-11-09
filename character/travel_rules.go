package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// TravelRules is the stateless service of the Travel rules (SRD 5.2.1, "Travel",
// p. 12). The zero value is ready to use. Safe for concurrent use.
type TravelRules struct{}

// Circumstances returns the Advantage or Disadvantage a travel pace puts on one
// ability check made while travelling, in a new slice (nil for none). The
// skill's default ability is assumed. Fast gives Disadvantage on Perception,
// Survival and Stealth; Normal on Stealth; Slow gives Advantage on Perception and
// Survival. A vehicle, an invalid pace and any other skill give none. Join the
// result with the roll's other circumstances through modifier.Resolve.
func (TravelRules) Circumstances(mode TravelMode, skill content.Skill) []modifier.RollCircumstance {
	land, ok := mode.(LandTravel)
	if !ok {
		return nil
	}
	wisdom := skill == content.SkillPerception || skill == content.SkillSurvival
	switch land.Pace {
	case PaceFast:
		if wisdom || skill == content.SkillStealth {
			return []modifier.RollCircumstance{modifier.Disadvantage{}}
		}
	case PaceNormal:
		if skill == content.SkillStealth {
			return []modifier.RollCircumstance{modifier.Disadvantage{}}
		}
	case PaceSlow:
		if wisdom {
			return []modifier.RollCircumstance{modifier.Advantage{}}
		}
	}
	return nil
}

// MilesPerHour returns the miles per hour of mode: the table figure of a pace, or
// the Speed of a vehicle. It does not scale by hours or days. An invalid mode
// gives 0.
func (TravelRules) MilesPerHour(mode TravelMode) float64 {
	switch m := mode.(type) {
	case LandTravel:
		return float64(m.Pace.Row().MilesPerHour())
	case VehicleTravel:
		return m.Speed.MilesPerHour()
	}
	return 0
}

// MountedHourMiles returns the miles covered in the one hour mounts may move
// "twice that distance" at pace: twice its miles per hour (SRD 5.2.1, "Travel
// Pace", p. 12). An invalid pace gives 0.
func (TravelRules) MountedHourMiles(pace TravelPace) int { return 2 * pace.Row().MilesPerHour() }
