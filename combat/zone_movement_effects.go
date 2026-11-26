package combat

import "github.com/Whealinghold/srd5e-go/content"

// ZoneMovementEffects are the movement effects a creature gets from the zones it
// is inside (persistent-spell-zones section 4.7): Difficult Terrain, a halved
// Speed and the extra movement cost per foot. The host passes them to movement;
// no Speed is changed here. The zero value is none.
type ZoneMovementEffects struct {
	difficultTerrain bool
	speedHalved      bool
	extraFeetPerFoot int
}

// DifficultTerrain reports whether Difficult Terrain applies. It is never cumulative.
func (m ZoneMovementEffects) DifficultTerrain() bool { return m.difficultTerrain }

// SpeedHalved reports whether the creature's Speed is halved. Two zones do not halve twice.
func (m ZoneMovementEffects) SpeedHalved() bool { return m.speedHalved }

// ExtraFeetPerFoot returns the largest extra movement per foot among the zones.
func (m ZoneMovementEffects) ExtraFeetPerFoot() int { return m.extraFeetPerFoot }

// None reports whether no zone changes the creature's movement.
func (m ZoneMovementEffects) None() bool { return m == ZoneMovementEffects{} }

// ExtraCostPerFoot returns the extra movement cost per foot moved: the larger of 1
// (when Difficult Terrain applies) and ExtraFeetPerFoot, never their sum (SRD
// 5.2.1, "Difficult Terrain", p. 181; "Wall of Thorns", p. 173).
func (m ZoneMovementEffects) ExtraCostPerFoot() int {
	cost := m.extraFeetPerFoot
	if m.difficultTerrain && cost < 1 {
		cost = 1
	}
	return cost
}

// ZoneMovementEffectsFor merges the movement effects of the zones a creature is in
// (persistent-spell-zones FR-044 to FR-048). Only active zones count where the
// creature is in the body region and is not designated. Difficult Terrain and a
// halved Speed are flags; the extra movement per foot is the largest. A creature
// outside every counted zone gets none.
func ZoneMovementEffectsFor(zones []PersistentZone, creature ZoneCreatureFacts) ZoneMovementEffects {
	var out ZoneMovementEffects
	for _, z := range zones {
		if z.ended || !creature.inRegion(content.DefaultZoneRegion) || z.exemptions.Contains(creature.creature) {
			continue
		}
		mv := z.facts.Movement()
		out.difficultTerrain = out.difficultTerrain || mv.DifficultTerrain()
		out.speedHalved = out.speedHalved || mv.SpeedHalved()
		out.extraFeetPerFoot = max(out.extraFeetPerFoot, mv.ExtraFeetPerFoot())
	}
	return out
}
