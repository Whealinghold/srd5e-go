package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// SpeciesUse is a species trait with limited uses (SRD 5.2.1, pp. 84-86). The
// zero value is invalid. Comparable and safe for concurrent use.
type SpeciesUse int

// The limited uses.
const (
	// BreathWeaponUse is the Dragonborn Breath Weapon.
	BreathWeaponUse SpeciesUse = iota + 1
	// DraconicFlightUse is the Dragonborn Draconic Flight.
	DraconicFlightUse
	// StonecunningUse is the Dwarf Stonecunning.
	StonecunningUse
	// GiantAncestryUse is the one chosen Giant Ancestry benefit of a Goliath.
	GiantAncestryUse
	// LargeFormUse is the Goliath Large Form.
	LargeFormUse
	// AdrenalineRushUse is the Orc Adrenaline Rush.
	AdrenalineRushUse
	// RelentlessEnduranceUse is the Orc Relentless Endurance.
	RelentlessEnduranceUse
	// LineageSpellThreeUse is the free cast of the level 3 spell of an Elven Lineage or Fiendish Legacy.
	LineageSpellThreeUse
	// LineageSpellFiveUse is the free cast of the level 5 spell of an Elven Lineage or Fiendish Legacy.
	LineageSpellFiveUse
	// GnomeSpellUse is the free casts of Speak with Animals of a Forest Gnome.
	GnomeSpellUse
)

// speciesUseCount is the number of SpeciesUse values.
const speciesUseCount = int(GnomeSpellUse)

// Valid reports whether u is one of the ten uses.
func (u SpeciesUse) Valid() bool { return u >= BreathWeaponUse && u <= GnomeSpellUse }

// String returns the snake_case name, or "unknown".
func (u SpeciesUse) String() string {
	return featNameOf(int(u), "breath_weapon", "draconic_flight", "stonecunning", "giant_ancestry", "large_form", "adrenaline_rush",
		"relentless_endurance", "lineage_spell_three", "lineage_spell_five", "gnome_spell")
}

// Limit returns how many times the use can be expended between renewals at a total
// level: the Proficiency Bonus for Breath Weapon, Stonecunning, Giant Ancestry,
// Adrenaline Rush and the Gnome spell, 1 for the others (table U). It is read at the
// moment of use, so a rise of Proficiency Bonus gives the extra use at once. It
// returns 0 for an invalid use.
func (u SpeciesUse) Limit(level Level) int {
	switch u {
	case BreathWeaponUse, StonecunningUse, GiantAncestryUse, AdrenalineRushUse, GnomeSpellUse:
		return ProficiencyBonusFor(level).Value()
	case DraconicFlightUse, LargeFormUse, RelentlessEnduranceUse, LineageSpellThreeUse, LineageSpellFiveUse:
		return 1
	}
	return 0
}

// MinimumLevel returns the character level from which the use exists (table U): 5
// for Draconic Flight, Large Form and the level 5 lineage spell, 3 for the level 3
// one, 1 for the others. It returns 0 for an invalid use.
func (u SpeciesUse) MinimumLevel() int {
	switch u {
	case DraconicFlightUse, LargeFormUse, LineageSpellFiveUse:
		return 5
	case LineageSpellThreeUse:
		return 3
	case BreathWeaponUse, StonecunningUse, GiantAncestryUse, AdrenalineRushUse, RelentlessEnduranceUse, GnomeSpellUse:
		return 1
	}
	return 0
}

// Renewal returns when the use is regained: finishing a Long Rest, or a Short or
// Long Rest for Adrenaline Rush. It returns the zero RechargeTrigger for an invalid use.
func (u SpeciesUse) Renewal() content.RechargeTrigger {
	switch {
	case u == AdrenalineRushUse:
		return content.RechargeShortOrLongRest
	case u.Valid():
		return content.RechargeLongRest
	}
	return 0
}

// SpeciesWindow is a moment a rest opens for the host to act on (SRD 5.2.1,
// "Resourceful", "High Elf", pp. 85-86). The zero value is invalid. Comparable.
type SpeciesWindow int

// The windows.
const (
	// WindowResourceful is the Human Resourceful window: the host grants Heroic Inspiration.
	WindowResourceful SpeciesWindow = iota + 1
	// WindowHighElfCantripReplacement is the High Elf window to replace the lineage cantrip.
	WindowHighElfCantripReplacement
)

// Valid reports whether w is one of the two windows.
func (w SpeciesWindow) Valid() bool {
	return w == WindowResourceful || w == WindowHighElfCantripReplacement
}

// String returns "resourceful", "high_elf_cantrip_replacement" or "unknown".
func (w SpeciesWindow) String() string {
	return featNameOf(int(w), "resourceful", "high_elf_cantrip_replacement")
}

// SpeciesTraitAction is the action a trait costs. The zero value is invalid. Comparable.
type SpeciesTraitAction int

// The actions.
const (
	// TraitBonusAction is a Bonus Action.
	TraitBonusAction SpeciesTraitAction = iota + 1
	// TraitReaction is a Reaction.
	TraitReaction
)

// String returns "bonus_action", "reaction" or "unknown".
func (a SpeciesTraitAction) String() string { return featNameOf(int(a), "bonus_action", "reaction") }

// BreathWeaponShape is the shape of a Breath Weapon, chosen each use (SRD 5.2.1,
// "Breath Weapon", p. 84). The zero value is invalid. Comparable.
type BreathWeaponShape int

// The shapes.
const (
	// BreathCone15 is a 15-foot Cone.
	BreathCone15 BreathWeaponShape = iota + 1
	// BreathLine30 is a 30-foot Line 5 feet wide.
	BreathLine30
)

// Valid reports whether s is one of the two shapes.
func (s BreathWeaponShape) Valid() bool { return s == BreathCone15 || s == BreathLine30 }

// String returns "cone_15", "line_30" or "unknown".
func (s BreathWeaponShape) String() string { return featNameOf(int(s), "cone_15", "line_30") }

// SpeciesUseRequest is a request to use a species trait, a closed set. Variants:
// BreathWeaponRequest, DraconicFlightRequest, StonecunningRequest,
// GiantAncestryRequest, StonesEnduranceRequest, RelentlessEnduranceRequest,
// LargeFormRequest, AdrenalineRushRequest, LineageSpellRequest, GnomeSpellRequest.
// StonesEnduranceRequest, RelentlessEnduranceRequest and the Stone GiantAncestryRequest
// exist so that SpeciesUseHasOwnUseCase is real: they belong to RollStonesEndurance
// and UseRelentlessEndurance.
type SpeciesUseRequest interface{ isSpeciesUseRequest() }

// BreathWeaponRequest asks to use Breath Weapon in Shape. Construct it with a keyed literal.
type BreathWeaponRequest struct{ Shape BreathWeaponShape }

// DraconicFlightRequest asks to use Draconic Flight.
type DraconicFlightRequest struct{}

// StonecunningRequest asks to use Stonecunning.
type StonecunningRequest struct{}

// GiantAncestryRequest asks to use the Giant Ancestry benefit Benefit. Construct it with a keyed literal.
type GiantAncestryRequest struct{ Benefit GiantAncestry }

// StonesEnduranceRequest names Stone's Endurance, which has its own use case.
type StonesEnduranceRequest struct{}

// RelentlessEnduranceRequest names Relentless Endurance, which has its own use case.
type RelentlessEnduranceRequest struct{}

// LargeFormRequest asks to use Large Form; HasEnoughSpace is the host's fact that
// the space is big enough. Construct it with a keyed literal.
type LargeFormRequest struct{ HasEnoughSpace bool }

// AdrenalineRushRequest asks to use Adrenaline Rush.
type AdrenalineRushRequest struct{}

// LineageSpellRequest asks for the free cast of the lineage or legacy spell of
// Level, 3 or 5. Construct it with a keyed literal.
type LineageSpellRequest struct{ Level int }

// GnomeSpellRequest asks for a free cast of Speak with Animals.
type GnomeSpellRequest struct{}

func (BreathWeaponRequest) isSpeciesUseRequest()        {}
func (DraconicFlightRequest) isSpeciesUseRequest()      {}
func (StonecunningRequest) isSpeciesUseRequest()        {}
func (GiantAncestryRequest) isSpeciesUseRequest()       {}
func (StonesEnduranceRequest) isSpeciesUseRequest()     {}
func (RelentlessEnduranceRequest) isSpeciesUseRequest() {}
func (LargeFormRequest) isSpeciesUseRequest()           {}
func (AdrenalineRushRequest) isSpeciesUseRequest()      {}
func (LineageSpellRequest) isSpeciesUseRequest()        {}
func (GnomeSpellRequest) isSpeciesUseRequest()          {}

// requestedUse returns the use a request draws on, and false for a nil request, an
// invalid shape, benefit or lineage level.
func requestedUse(r SpeciesUseRequest) (SpeciesUse, bool) {
	switch v := r.(type) {
	case BreathWeaponRequest:
		return BreathWeaponUse, v.Shape.Valid()
	case DraconicFlightRequest:
		return DraconicFlightUse, true
	case StonecunningRequest:
		return StonecunningUse, true
	case GiantAncestryRequest:
		return GiantAncestryUse, v.Benefit.Valid()
	case StonesEnduranceRequest:
		return GiantAncestryUse, true
	case RelentlessEnduranceRequest:
		return RelentlessEnduranceUse, true
	case LargeFormRequest:
		return LargeFormUse, true
	case AdrenalineRushRequest:
		return AdrenalineRushUse, true
	case LineageSpellRequest:
		if v.Level == 5 {
			return LineageSpellFiveUse, true
		}
		return LineageSpellThreeUse, v.Level == 3
	case GnomeSpellRequest:
		return GnomeSpellUse, true
	}
	return 0, false
}
