package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// PrismaticWallState is how many layers of a Prismatic Wall have been destroyed, always from red onward (SRD 5.2.1,
// "Prismatic Wall", p. 155). The zero value is the intact wall. Immutable and comparable.
type PrismaticWallState struct{ destroyed int }

// NewPrismaticWallStateAfter returns the wall after count layers were destroyed. It returns an error matching
// ErrWallDestroyedCount for a count outside 0 to 7. It never panics.
func NewPrismaticWallStateAfter(count int) (PrismaticWallState, error) {
	if count < 0 || count > 7 {
		return PrismaticWallState{}, ErrWallDestroyedCount
	}
	return PrismaticWallState{count}, nil
}

// Destroyed returns the number of destroyed layers.
func (w PrismaticWallState) Destroyed() int { return w.destroyed }

// Layers returns the colours still present, in order from red, as a new slice; it is empty for a wall with no layer
// left.
func (w PrismaticWallState) Layers() []PrismaticColor {
	var out []PrismaticColor
	for c := PrismaticColor(w.destroyed + 1); c <= PrismaticViolet; c++ {
		out = append(out, c)
	}
	return out
}

// PrismaticLayerDestroyer is the closed set of things that can destroy a layer (SRD 5.2.1, "Prismatic Layers",
// p. 155). Variants: DamageToLayer, StrongWind, PortalOpened, DaylightBrightLight, DispelMagicOnLayer. There is no
// value for Antimagic Field, which has no effect on the wall.
type PrismaticLayerDestroyer interface{ isPrismaticLayerDestroyer() }

// DamageToLayer is one instance of damage of a type: 25 Cold destroys red, 60 Force yellow and 25 Fire blue. Build
// it with a keyed literal.
type DamageToLayer struct {
	Type   content.DamageType
	Amount int
}

// StrongWind destroys the orange layer.
type StrongWind struct{}

// PortalOpened is Passwall or an equal or higher level portal spell of SpellLevel; level 5 or more destroys green.
// Build it with a keyed literal.
type PortalOpened struct{ SpellLevel int }

// DaylightBrightLight is the Bright Light of Daylight, which destroys indigo.
type DaylightBrightLight struct{}

// DispelMagicOnLayer is Dispel Magic, which destroys violet only.
type DispelMagicOnLayer struct{}

func (DamageToLayer) isPrismaticLayerDestroyer()       {}
func (StrongWind) isPrismaticLayerDestroyer()          {}
func (PortalOpened) isPrismaticLayerDestroyer()        {}
func (DaylightBrightLight) isPrismaticLayerDestroyer() {}
func (DispelMagicOnLayer) isPrismaticLayerDestroyer()  {}

// PrismaticNotDestroyedReason is why a layer was not destroyed. The zero value is invalid.
type PrismaticNotDestroyedReason int

const (
	// LayerNotOutermost: a layer is destroyed only after the ones before it, from red to violet.
	LayerNotOutermost PrismaticNotDestroyedReason = iota + 1
	// LayerAlreadyDestroyed: the layer is already gone.
	LayerAlreadyDestroyed
	// LayerWrongDestroyer: the destroyer is not the kind that destroys this layer.
	LayerWrongDestroyer
	// LayerBelowThreshold: the right kind of damage or portal, but less than the layer needs.
	LayerBelowThreshold
)

// String returns "not_outermost", "already_destroyed", "wrong_destroyer", "below_threshold" or "unknown".
func (r PrismaticNotDestroyedReason) String() string {
	return enumName([]string{"not_outermost", "already_destroyed", "wrong_destroyer", "below_threshold"}, int(r))
}

// PrismaticLayerDestruction is the closed set of results of Destroy. Variants: PrismaticLayerDestroyed,
// PrismaticLayerNotDestroyed.
type PrismaticLayerDestruction interface{ isPrismaticLayerDestruction() }

// PrismaticLayerDestroyed: the layer is gone.
type PrismaticLayerDestroyed struct{ Layer PrismaticColor }

// PrismaticLayerNotDestroyed: nothing changed.
type PrismaticLayerNotDestroyed struct{ Reason PrismaticNotDestroyedReason }

func (PrismaticLayerDestroyed) isPrismaticLayerDestruction()    {}
func (PrismaticLayerNotDestroyed) isPrismaticLayerDestruction() {}

// destroyerFits returns 0 when d destroys layer, else LayerWrongDestroyer or LayerBelowThreshold (table W of
// spell-outcome-tables.md). The amount of a DamageToLayer is one damage instance (Q-5).
func destroyerFits(layer PrismaticColor, d PrismaticLayerDestroyer) PrismaticNotDestroyedReason {
	damage := func(t content.DamageType, least int) PrismaticNotDestroyedReason {
		v, ok := d.(DamageToLayer)
		switch {
		case !ok || v.Type != t:
			return LayerWrongDestroyer
		case v.Amount < least:
			return LayerBelowThreshold
		}
		return 0
	}
	switch layer {
	case PrismaticRed:
		return damage(content.DamageCold, 25)
	case PrismaticYellow:
		return damage(content.DamageForce, 60)
	case PrismaticBlue:
		return damage(content.DamageFire, 25)
	case PrismaticOrange:
		if _, ok := d.(StrongWind); !ok {
			return LayerWrongDestroyer
		}
	case PrismaticGreen:
		p, ok := d.(PortalOpened)
		switch {
		case !ok:
			return LayerWrongDestroyer
		case p.SpellLevel < 5:
			return LayerBelowThreshold
		}
	case PrismaticIndigo:
		if _, ok := d.(DaylightBrightLight); !ok {
			return LayerWrongDestroyer
		}
	case PrismaticViolet:
		if _, ok := d.(DispelMagicOnLayer); !ok {
			return LayerWrongDestroyer
		}
	}
	return 0
}

// Destroy tries to destroy layer with d. A layer goes only when it is the outermost one still present and d is its
// destroyer, and then the new state has one more destroyed layer; otherwise the state is returned unchanged with the
// reason (SRD 5.2.1, "Prismatic Wall", p. 155). A layer that is not a colour is not the outermost.
func (w PrismaticWallState) Destroy(layer PrismaticColor, d PrismaticLayerDestroyer) (PrismaticWallState, PrismaticLayerDestruction) {
	order := layer.Order()
	switch {
	case order >= 1 && order <= w.destroyed:
		return w, PrismaticLayerNotDestroyed{LayerAlreadyDestroyed}
	case order != w.destroyed+1:
		return w, PrismaticLayerNotDestroyed{LayerNotOutermost}
	}
	if r := destroyerFits(layer, d); r != 0 {
		return w, PrismaticLayerNotDestroyed{r}
	}
	return PrismaticWallState{w.destroyed + 1}, PrismaticLayerDestroyed{layer}
}

// PrismaticLayerCrossing is what one layer did to a creature crossing the wall: its Dexterity save and the strike,
// if the layer has one for that save. Immutable.
type PrismaticLayerCrossing struct {
	color  PrismaticColor
	save   UnarmedStrikeSave
	strike PrismaticStrike
}

// Color returns the layer.
func (c PrismaticLayerCrossing) Color() PrismaticColor { return c.color }

// Save returns the Dexterity save against this layer.
func (c PrismaticLayerCrossing) Save() UnarmedStrikeSave { return c.save }

// Strike returns the strike of the layer and true, or false when there is none (an indigo or violet layer after a
// successful save).
func (c PrismaticLayerCrossing) Strike() (PrismaticStrike, bool) { return c.strike, c.strike != nil }

// PrismaticWallCrossing is what a wall did to one creature crossing it: the layers still present, in order. It is
// empty for a wall with no layer left. Whether the creature keeps moving after a strike is not modelled. Immutable.
type PrismaticWallCrossing struct {
	// Creature is the crossing creature.
	Creature CombatantID
	layers   []PrismaticLayerCrossing
}

// Layers returns the layers crossed, in order from red, as a new slice.
func (c PrismaticWallCrossing) Layers() []PrismaticLayerCrossing { return slices.Clone(c.layers) }
