package combat

import (
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// SaveSourceKind says what a saving throw is against, as the host classifies it. The
// zero value is invalid. Comparable and safe for concurrent use.
type SaveSourceKind int

const (
	// SpellOrMagicalEffect: the saving throw is against a spell or another magical effect.
	SpellOrMagicalEffect SaveSourceKind = iota + 1
	// OtherSaveSource: the saving throw is against anything else.
	OtherSaveSource
)

// Valid reports whether k is one of the two kinds.
func (k SaveSourceKind) Valid() bool { return k == SpellOrMagicalEffect || k == OtherSaveSource }

// String returns "spell_or_magical_effect", "other" or "unknown".
func (k SaveSourceKind) String() string {
	return enumName([]string{"spell_or_magical_effect", "other"}, int(k))
}

// MagicResistanceCircumstances returns the circumstances Magic Resistance adds to a
// saving throw against source (SRD 5.2.1, "Balor", p. 261): [modifier.Advantage{}] for
// SpellOrMagicalEffect, an empty list for anything else, for a zero trait and for an
// invalid source. The list feeds modifier.Resolve unchanged, so a Disadvantage from
// another source still cancels it (SRD 5.2.1, "They Don't Stack", p. 8). It never fails.
func MagicResistanceCircumstances(t MagicResistanceTrait, source SaveSourceKind) []modifier.RollCircumstance {
	if t.IsZero() || source != SpellOrMagicalEffect {
		return []modifier.RollCircumstance{}
	}
	return []modifier.RollCircumstance{modifier.Advantage{}}
}

// AllyFact is what the host knows of one ally of the monster for Pack Tactics: the
// distance in feet from the ally to the creature being attacked (the target, not the
// monster) and whether the ally has the Incapacitated condition, which Paralyzed,
// Petrified, Stunned and Unconscious include. Construct with a keyed literal.
type AllyFact struct {
	DistanceFeet  int
	Incapacitated bool
}

// PackTacticsCircumstances returns [modifier.Advantage{}] when at least one ally is 0 to 5
// feet inclusive from the creature being attacked and is not Incapacitated, and an empty
// list otherwise, also for a zero trait (SRD 5.2.1, "Wolf", p. 364; "Incapacitated",
// p. 184). The host lists the allies of the monster, not the monster itself.
//
// It returns an error matching srderr.ErrInvalidInput, and no circumstances, for an ally
// at a negative distance, and a LimitExceededError (plan P10) for more than 256 allies. It never panics.
func PackTacticsCircumstances(t PackTacticsTrait, allies []AllyFact) ([]modifier.RollCircumstance, error) {
	if err := overCap("pack tactics allies", len(allies), maxPackTacticsAllies); err != nil {
		return nil, err
	}
	flanked := false
	for _, a := range allies {
		if a.DistanceFeet < 0 {
			return nil, invalidInput("an ally cannot be at a negative distance")
		}
		flanked = flanked || (a.DistanceFeet <= 5 && !a.Incapacitated)
	}
	if t.IsZero() || !flanked {
		return []modifier.RollCircumstance{}, nil
	}
	return []modifier.RollCircumstance{modifier.Advantage{}}, nil
}

// SunlightPenaltyCircumstances returns [modifier.Disadvantage{}] when inSunlight and kind is
// a D20 Test the trait's scope covers, and an empty list otherwise, also for a zero trait
// or a kind that is not valid (SRD 5.2.1, "Drider", p. 281; "Shadow", p. 322). Sunlight
// Sensitivity covers ability checks and attack rolls, not saving throws; Sunlight Weakness
// covers all three. It never fails.
func SunlightPenaltyCircumstances(t SunlightPenaltyTrait, kind d20.Kind, inSunlight bool) []modifier.RollCircumstance {
	covered := kind == d20.AbilityCheck || kind == d20.AttackRoll || (kind == d20.SavingThrow && t.scope == SunlightPenaltyAllD20Tests)
	if t.IsZero() || !inSunlight || !covered {
		return []modifier.RollCircumstance{}
	}
	return []modifier.RollCircumstance{modifier.Disadvantage{}}
}
