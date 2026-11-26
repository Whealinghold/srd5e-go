package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Speed arithmetic of a heavy load and of armor (movement-lifting-and-flight.md
// section 4.1). Pure and silent: the limit and the penalty come from the character
// context, and this file only applies them to MovementSpeeds.

// LoadLimitScope says which speeds a load limit reaches. The zero value is
// LoadLimitScopeAllSpeedsByEqualAmount.
type LoadLimitScope int

// The two scopes.
const (
	// LoadLimitScopeAllSpeedsByEqualAmount treats the cap as a decrease of the Speed and
	// carries it to every special speed (SRD 5.2.1, "Changes to Your Speeds", p. 188).
	LoadLimitScopeAllSpeedsByEqualAmount LoadLimitScope = iota
	// LoadLimitScopeWalkOnly changes the walking Speed alone, for a host that rules the cap
	// is not a decrease.
	LoadLimitScopeWalkOnly
)

// LimitSpeedsByLoad returns speeds with the walking Speed capped at limit.LimitFeet (SRD
// 5.2.1, "Carrying Capacity", p. 178). By default the cap is a decrease of the walking
// Speed that every special speed the creature has follows by an equal amount, none falling
// below 0; with LoadLimitScopeWalkOnly only the walking Speed changes. A Speed at or below
// the cap is unchanged: the cap never raises a Speed. It accepts only the
// character.LoadSpeedLimited assessment, so a load that does not limit the Speed is not an
// input. speeds is not altered. It cannot fail and never panics.
func LimitSpeedsByLoad(speeds MovementSpeeds, limit character.LoadSpeedLimited, scope LoadLimitScope) MovementSpeeds {
	capped := min(speeds.Walk(), limit.LimitFeet())
	if scope == LoadLimitScopeWalkOnly {
		out := speeds
		out.feet[0] = capped
		return out
	}
	return speeds.AdjustedBy(-(speeds.Walk() - capped))
}

// ApplyArmorSpeedPenalty returns speeds less the Speed penalty of armor worn by a creature
// of the given Strength score, a decrease every special speed follows by an equal amount,
// none falling below 0 (SRD 5.2.1, "Armor", Strength, p. 92). When a heavy load also limits
// the Speed, apply the armor first and LimitSpeedsByLoad to the result. speeds is not
// altered. It cannot fail and never panics.
func ApplyArmorSpeedPenalty(speeds MovementSpeeds, armor content.Armor, strength ability.Score) MovementSpeeds {
	return speeds.AdjustedBy(-character.ArmorRules{}.SpeedPenaltyFeet(armor, strength))
}
