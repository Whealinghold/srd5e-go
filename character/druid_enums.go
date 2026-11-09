package character

import "github.com/Whealinghold/srd5e-go/content"

// PrimalOrder is the sacred role a Druid chose at level 1 (SRD 5.2.1, "Level 1: Primal Order", p. 42). The zero value is
// no order; it is comparable.
type PrimalOrder int

const (
	// PrimalOrderMagician adds one cantrip known and a bonus to Intelligence (Arcana or Nature) checks.
	PrimalOrderMagician PrimalOrder = iota + 1
	// PrimalOrderWarden grants Martial weapon proficiency and Medium armor training.
	PrimalOrderWarden
)

// String returns "magician", "warden" or "none" for the zero value.
func (o PrimalOrder) String() string {
	switch o {
	case PrimalOrderMagician:
		return "magician"
	case PrimalOrderWarden:
		return "warden"
	}
	return "none"
}

// GrantsMartialWeapons reports whether the order grants proficiency with Martial weapons: the Warden does, the Magician does not.
func (o PrimalOrder) GrantsMartialWeapons() bool { return o == PrimalOrderWarden }

// GrantsMediumArmor reports whether the order grants training with Medium armor: the Warden does, the Magician does not.
func (o PrimalOrder) GrantsMediumArmor() bool { return o == PrimalOrderWarden }

// Valid reports whether o is Magician or Warden.
func (o PrimalOrder) Valid() bool { return o == PrimalOrderMagician || o == PrimalOrderWarden }

// ElementalFury is the option a Druid chose at level 7 (SRD 5.2.1, "Level 7: Elemental Fury", p. 43). The zero value is no
// option; it is comparable.
type ElementalFury int

const (
	// ElementalFuryPotentSpellcasting adds the Wisdom modifier to Druid cantrip damage and, from level 15, lengthens their range.
	ElementalFuryPotentSpellcasting ElementalFury = iota + 1
	// ElementalFuryPrimalStrike adds 1d8 (2d8 from level 15) Cold, Fire, Lightning or Thunder damage to a hit.
	ElementalFuryPrimalStrike
)

// String returns "potent_spellcasting", "primal_strike" or "none" for the zero value.
func (o ElementalFury) String() string {
	switch o {
	case ElementalFuryPotentSpellcasting:
		return "potent_spellcasting"
	case ElementalFuryPrimalStrike:
		return "primal_strike"
	}
	return "none"
}

// Valid reports whether o is one of the two options.
func (o ElementalFury) Valid() bool {
	return o == ElementalFuryPotentSpellcasting || o == ElementalFuryPrimalStrike
}

// LandType is a type of land of the Circle of the Land (SRD 5.2.1, "Level 3: Circle of the Land Spells", p. 46). The zero
// value is no land; it is comparable.
type LandType int

const (
	// LandArid is the arid land: Fire Resistance.
	LandArid LandType = iota + 1
	// LandPolar is the polar land: Cold Resistance.
	LandPolar
	// LandTemperate is the temperate land: Lightning Resistance.
	LandTemperate
	// LandTropical is the tropical land: Poison Resistance.
	LandTropical
)

func landTypeNames() []string { return []string{"arid", "polar", "temperate", "tropical"} }

// String returns "arid", "polar", "temperate", "tropical" or "none" for the zero value.
func (l LandType) String() string {
	if !l.Valid() {
		return "none"
	}
	return landTypeNames()[l-1]
}

// Valid reports whether l is one of the four lands.
func (l LandType) Valid() bool { return l >= LandArid && l <= LandTropical }

// DruidActionCost is the action a Druid feature costs (SRD 5.2.1, pp. 42-46). The zero value is DruidActionCostNone.
type DruidActionCost int

const (
	// DruidActionCostNone: no action is required.
	DruidActionCostNone DruidActionCost = iota
	// DruidActionCostMagic: a Magic action.
	DruidActionCostMagic
	// DruidActionCostBonus: a Bonus Action.
	DruidActionCostBonus
)

// String returns "none", "magic_action" or "bonus_action".
func (c DruidActionCost) String() string {
	switch c {
	case DruidActionCostNone:
		return "none"
	case DruidActionCostMagic:
		return "magic_action"
	case DruidActionCostBonus:
		return "bonus_action"
	}
	return "unknown"
}

// RestWindowPermission is a choice that a finished rest opens until the next turn, initiative or rest (class-druid.md D-4).
type RestWindowPermission int

const (
	// RestWindowLandTypeChange: the land type may be chosen (a Long Rest, with the Circle of the Land).
	RestWindowLandTypeChange RestWindowPermission = iota + 1
	// RestWindowKnownFormReplacement: a known form may be replaced (a Long Rest, with Wild Shape).
	RestWindowKnownFormReplacement
	// RestWindowNaturalRecoveryRecovery: Natural Recovery may recover slots (a Short Rest, with Natural Recovery).
	RestWindowNaturalRecoveryRecovery
)

// String returns "land_type_change", "known_form_replacement", "natural_recovery_recovery" or "unknown".
func (p RestWindowPermission) String() string {
	switch p {
	case RestWindowLandTypeChange:
		return "land_type_change"
	case RestWindowKnownFormReplacement:
		return "known_form_replacement"
	case RestWindowNaturalRecoveryRecovery:
		return "natural_recovery_recovery"
	}
	return "unknown"
}

// Valid reports whether p is one of the three permissions.
func (p RestWindowPermission) Valid() bool {
	return p >= RestWindowLandTypeChange && p <= RestWindowNaturalRecoveryRecovery
}

// EquipmentDisposition says what happens to an item when a Druid assumes a Beast form (SRD 5.2.1, "Rules While
// Shape-Shifted", p. 43). The zero value is unassigned.
type EquipmentDisposition int

const (
	// EquipmentDispositionFallsInSpace: the item falls in the Druid's space.
	EquipmentDispositionFallsInSpace EquipmentDisposition = iota + 1
	// EquipmentDispositionMergesIntoForm: the item merges into the form and has no effect.
	EquipmentDispositionMergesIntoForm
	// EquipmentDispositionWorn: the form wears the item and it functions as normal.
	EquipmentDispositionWorn
)

// DruidItemKey is the host's opaque key of a worn or carried item (class-druid.md FR-084).
type DruidItemKey string

// DruidItemPlan is what the host decided for one item on assuming a Beast form. Construct it with a keyed literal.
type DruidItemPlan struct {
	Key         DruidItemKey
	Disposition EquipmentDisposition
	// FormCanWear is the host's statement that the form can wear the item; it matters for EquipmentDispositionWorn.
	FormCanWear bool
}

// WildShapeEndReason is why Wild Shape ended (SRD 5.2.1, "Level 2: Wild Shape", p. 42). The zero value is not a reason.
type WildShapeEndReason int

const (
	// WildShapeEndDurationElapsed: the hours elapsed.
	WildShapeEndDurationElapsed WildShapeEndReason = iota + 1
	// WildShapeEndIncapacitated: the Druid has the Incapacitated condition.
	WildShapeEndIncapacitated
	// WildShapeEndDied: the Druid died.
	WildShapeEndDied
	// WildShapeEndUsedWildShapeAgain: Wild Shape was used again (produced only by a switch of forms).
	WildShapeEndUsedWildShapeAgain
	// WildShapeEndLeftEarly: the Druid left the form as a Bonus Action.
	WildShapeEndLeftEarly
)

// String returns the snake_case name or "unknown".
func (r WildShapeEndReason) String() string {
	switch r {
	case WildShapeEndDurationElapsed:
		return "duration_elapsed"
	case WildShapeEndIncapacitated:
		return "incapacitated"
	case WildShapeEndDied:
		return "died"
	case WildShapeEndUsedWildShapeAgain:
		return "used_wild_shape_again"
	case WildShapeEndLeftEarly:
		return "left_early"
	}
	return "unknown"
}

// SanctuaryEndReason is why a Nature's Sanctuary Cube ended (SRD 5.2.1, "Level 14: Nature's Sanctuary", p. 46). The zero
// value is not a reason.
type SanctuaryEndReason int

const (
	// SanctuaryEndMinuteElapsed: the minute elapsed.
	SanctuaryEndMinuteElapsed SanctuaryEndReason = iota + 1
	// SanctuaryEndIncapacitated: the Druid has the Incapacitated condition.
	SanctuaryEndIncapacitated
	// SanctuaryEndDied: the Druid died.
	SanctuaryEndDied
)

// String returns the snake_case name or "unknown".
func (r SanctuaryEndReason) String() string {
	switch r {
	case SanctuaryEndMinuteElapsed:
		return "minute_elapsed"
	case SanctuaryEndIncapacitated:
		return "incapacitated"
	case SanctuaryEndDied:
		return "died"
	}
	return "unknown"
}

// PrimalStrikeAttackSource is the kind of attack a Primal Strike is tried on (SRD 5.2.1, "Level 7: Elemental Fury", p. 43).
// The zero value is not a source.
type PrimalStrikeAttackSource int

const (
	// PrimalStrikeSourceWeapon: an attack roll using a weapon, in or out of Wild Shape.
	PrimalStrikeSourceWeapon PrimalStrikeAttackSource = iota + 1
	// PrimalStrikeSourceBeastForm: a Beast form's attack in Wild Shape.
	PrimalStrikeSourceBeastForm
	// PrimalStrikeSourceSpell: a spell attack, which Primal Strike does not apply to.
	PrimalStrikeSourceSpell
	// PrimalStrikeSourceOther: any other attack, which Primal Strike does not apply to.
	PrimalStrikeSourceOther
)

// Valid reports whether s is one of the four sources.
func (s PrimalStrikeAttackSource) Valid() bool {
	return s >= PrimalStrikeSourceWeapon && s <= PrimalStrikeSourceOther
}

// BeastFormFacts are the facts about a Beast stat block that the Wild Shape rules read (class-druid.md FR-171). Immutable
// and comparable with ==; the zero value has no id and is invalid.
type BeastFormFacts struct {
	id       content.MonsterID
	kind     content.CreatureType
	rating   content.ChallengeRating
	flySpeed bool
}

// NewBeastFormFacts returns the facts of the form id. It returns an error matching srderr.ErrInvalidInput for an invalid
// id. It never panics.
func NewBeastFormFacts(id content.MonsterID, kind content.CreatureType, rating content.ChallengeRating, hasFlySpeed bool) (BeastFormFacts, error) {
	if !id.Valid() {
		return BeastFormFacts{}, invalidInput("Beast form with an invalid monster id")
	}
	return BeastFormFacts{id: id, kind: kind, rating: rating, flySpeed: hasFlySpeed}, nil
}

// BeastFormFactsFromMonster maps a content Monster to the facts the rules read: its id, creature type, Challenge Rating
// and whether any of its speeds has the fly mode.
func BeastFormFactsFromMonster(m content.Monster) BeastFormFacts {
	fly := false
	for _, s := range m.Speeds() {
		for _, mode := range s.Modes() {
			fly = fly || mode == content.SpeedFly
		}
	}
	return BeastFormFacts{id: m.ID(), kind: m.Type(), rating: m.ChallengeRating(), flySpeed: fly}
}

// ID returns the monster id of the form.
func (f BeastFormFacts) ID() content.MonsterID { return f.id }

// CreatureType returns the creature type.
func (f BeastFormFacts) CreatureType() content.CreatureType { return f.kind }

// ChallengeRating returns the Challenge Rating.
func (f BeastFormFacts) ChallengeRating() content.ChallengeRating { return f.rating }

// HasFlySpeed reports whether the form has a Fly Speed.
func (f BeastFormFacts) HasFlySpeed() bool { return f.flySpeed }
