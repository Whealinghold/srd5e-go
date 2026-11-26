package combat

// HazardKind is one of the five hazards of SRD 5.2.1, "Hazard", p. 182. Its
// String is the value of srd5e.combat.hazard.kind. The zero value is invalid.
type HazardKind int

// The five hazards, in alphabetical order.
const (
	// HazardBurning is the Burning hazard (p. 178).
	HazardBurning HazardKind = iota + 1
	// HazardDehydration is the Dehydration hazard (p. 181).
	HazardDehydration
	// HazardFalling is the Falling hazard (p. 182).
	HazardFalling
	// HazardMalnutrition is the Malnutrition hazard (p. 185).
	HazardMalnutrition
	// HazardSuffocation is the Suffocation hazard (p. 189).
	HazardSuffocation
)

// Valid reports whether k is one of the five hazards.
func (k HazardKind) Valid() bool { return k >= HazardBurning && k <= HazardSuffocation }

// String returns the lowercase name of k, or "unknown".
func (k HazardKind) String() string {
	return enumName([]string{"burning", "dehydration", "falling", "malnutrition", "suffocation"}, int(k))
}

// HazardExhaustionOutcome is what a hazard did to a creature's Exhaustion: a
// projection of the condition result, plus NoExhaustion for a call on which none
// was applied. The zero value is invalid.
type HazardExhaustionOutcome int

// The outcomes.
const (
	// HazardNoExhaustion: no Exhaustion was applied.
	HazardNoExhaustion HazardExhaustionOutcome = iota + 1
	// HazardExhaustionGained: one Exhaustion level was added.
	HazardExhaustionGained
	// HazardExhaustionAtMaximum: the creature was already at level 6.
	HazardExhaustionAtMaximum
	// HazardImmune: the creature is immune to Exhaustion.
	HazardImmune
)

// String returns the snake_case name of o (the srd5e.combat.result value), or "unknown".
func (o HazardExhaustionOutcome) String() string {
	return enumName([]string{"no_exhaustion", "exhaustion_gained", "exhaustion_at_maximum", "immune"}, int(o))
}
