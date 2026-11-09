package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

// DivineOrder is the sacred role a Cleric chose at level 1 (SRD 5.2.1, "Level 1:
// Divine Order", p. 37). The zero value is no order; it is comparable.
type DivineOrder int

const (
	// DivineOrderProtector grants Martial weapon proficiency and Heavy armor training.
	DivineOrderProtector DivineOrder = iota + 1
	// DivineOrderThaumaturge grants one extra cantrip and a bonus to Arcana and Religion checks.
	DivineOrderThaumaturge
)

// String returns "protector", "thaumaturge", or "none" for the zero value.
func (o DivineOrder) String() string {
	switch o {
	case DivineOrderProtector:
		return "protector"
	case DivineOrderThaumaturge:
		return "thaumaturge"
	}
	return "none"
}

// BlessedStrikesOption is the option a Cleric chose at level 7 (SRD 5.2.1, "Level
// 7: Blessed Strikes", p. 38). The zero value is no option; it is comparable.
type BlessedStrikesOption int

const (
	// BlessedStrikesDivineStrike adds 1d8 (2d8 from level 14) Necrotic or Radiant damage to a weapon hit.
	BlessedStrikesDivineStrike BlessedStrikesOption = iota + 1
	// BlessedStrikesPotentSpellcasting adds the Wisdom modifier to Cleric cantrip damage.
	BlessedStrikesPotentSpellcasting
)

// String returns "divine_strike", "potent_spellcasting", or "none" for the zero value.
func (o BlessedStrikesOption) String() string {
	switch o {
	case BlessedStrikesDivineStrike:
		return "divine_strike"
	case BlessedStrikesPotentSpellcasting:
		return "potent_spellcasting"
	}
	return "none"
}

// ClericChoices is the Cleric's chosen Divine Order (with the Thaumaturge's extra
// cantrip) and Blessed Strikes option. Neither can change once chosen (class-cleric.md
// D-4, D-7). It is a FeatureState stored under ClericChoicesKey; an absent state is
// the empty choices. Immutable, comparable with ==, and the zero value is the empty
// choices. Safe for concurrent use.
type ClericChoices struct {
	order        DivineOrder
	extraCantrip content.SpellID
	strikes      BlessedStrikesOption
}

// ClericChoicesKey returns the FeatureStateKey named cleric_choices.
func ClericChoicesKey() FeatureStateKey[ClericChoices] {
	return MustFeatureStateKey[ClericChoices]("cleric_choices")
}

// NewClericChoices returns the choices with the given order, extra cantrip ("" for
// none) and option. It returns an error matching srderr.ErrInvalidInput for an
// order or option outside the closed sets, or an invalid non-empty cantrip id. It
// never panics.
func NewClericChoices(order DivineOrder, extraCantrip content.SpellID, strikes BlessedStrikesOption) (ClericChoices, error) {
	if order < 0 || order > DivineOrderThaumaturge || strikes < 0 || strikes > BlessedStrikesPotentSpellcasting ||
		extraCantrip != "" && !extraCantrip.Valid() {
		return ClericChoices{}, invalidInput("Cleric choices outside the closed sets")
	}
	return ClericChoices{order: order, extraCantrip: extraCantrip, strikes: strikes}, nil
}

// DivineOrder returns the chosen order, or the zero value for none.
func (c ClericChoices) DivineOrder() DivineOrder { return c.order }

// ExtraCantrip returns the Thaumaturge's extra cantrip, or "" for none.
func (c ClericChoices) ExtraCantrip() content.SpellID { return c.extraCantrip }

// BlessedStrikes returns the chosen option, or the zero value for none.
func (c ClericChoices) BlessedStrikes() BlessedStrikesOption { return c.strikes }

// ExtraCantrips returns the extra-cantrip allowance for spellcasting: 1 for a
// Thaumaturge, else 0 (class-cleric.md FR-034, FR-141).
func (c ClericChoices) ExtraCantrips() int {
	if c.order == DivineOrderThaumaturge {
		return 1
	}
	return 0
}

// Equal reports whether other is a ClericChoices with the same choices.
func (c ClericChoices) Equal(other FeatureState) bool {
	o, ok := other.(ClericChoices)
	return ok && o == c
}

// DivineInterventionState is whether Divine Intervention can be used, a closed set.
// Variants: DivineInterventionAvailable, DivineInterventionUnavailableUntilLongRest,
// DivineInterventionRecharging.
type DivineInterventionState interface{ isDivineInterventionState() }

// DivineInterventionAvailable: Divine Intervention can be used.
type DivineInterventionAvailable struct{}

// DivineInterventionUnavailableUntilLongRest: used; a finished Long Rest makes it available again.
type DivineInterventionUnavailableUntilLongRest struct{}

// DivineInterventionRecharging: Wish was chosen; it is unavailable until LongRestsRemaining
// more Long Rests are finished (2 to 8 for the 2d4 default). Construct it with a keyed literal.
type DivineInterventionRecharging struct{ LongRestsRemaining int }

func (DivineInterventionAvailable) isDivineInterventionState()                {}
func (DivineInterventionUnavailableUntilLongRest) isDivineInterventionState() {}
func (DivineInterventionRecharging) isDivineInterventionState()               {}

// ClericResources is the Cleric's Channel Divinity uses (maximum and remaining)
// and the Divine Intervention state. It holds the maximum, so no Cleric operation
// loads content (class-cleric.md FR-044). It is a FeatureState stored under
// ClericResourcesKey. Immutable, comparable with ==; the zero value is 0 of 0 uses
// with Divine Intervention available. Safe for concurrent use.
type ClericResources struct {
	channelMax, channelRemaining int
	intervention                 DivineInterventionState
}

// ClericResourcesKey returns the FeatureStateKey named cleric_resources.
func ClericResourcesKey() FeatureStateKey[ClericResources] {
	return MustFeatureStateKey[ClericResources]("cleric_resources")
}

// InitialClericResources returns maximum uses of Channel Divinity, all remaining,
// with Divine Intervention available. A negative maximum is 0.
func InitialClericResources(maximum int) ClericResources {
	maximum = max(maximum, 0)
	return ClericResources{channelMax: maximum, channelRemaining: maximum, intervention: DivineInterventionAvailable{}}
}

// NewClericResources returns the resources. It returns an error matching
// srderr.ErrInvalidInput for a negative maximum, a remaining count outside 0 to the
// maximum (class-cleric.md FR-047), or a nil or invalid Divine Intervention state
// (a Recharging count below 1). It never panics.
func NewClericResources(maximum, remaining int, intervention DivineInterventionState) (ClericResources, error) {
	if maximum < 0 || remaining < 0 || remaining > maximum {
		return ClericResources{}, invalidInput("Cleric Channel Divinity " + strconv.Itoa(remaining) + " of " + strconv.Itoa(maximum) + " uses")
	}
	switch s := intervention.(type) {
	case DivineInterventionAvailable, DivineInterventionUnavailableUntilLongRest:
	case DivineInterventionRecharging:
		if s.LongRestsRemaining < 1 {
			return ClericResources{}, invalidInput("Divine Intervention recharging with fewer than 1 Long Rest to go")
		}
	default:
		return ClericResources{}, invalidInput("Divine Intervention state is not one of the closed set")
	}
	return ClericResources{channelMax: maximum, channelRemaining: remaining, intervention: intervention}, nil
}

// ChannelDivinityMaximum returns the maximum Channel Divinity uses.
func (r ClericResources) ChannelDivinityMaximum() int { return r.channelMax }

// ChannelDivinityRemaining returns the unexpended Channel Divinity uses.
func (r ClericResources) ChannelDivinityRemaining() int { return r.channelRemaining }

// DivineIntervention returns the Divine Intervention state (Available for the zero value).
func (r ClericResources) DivineIntervention() DivineInterventionState {
	if r.intervention == nil {
		return DivineInterventionAvailable{}
	}
	return r.intervention
}

// Equal reports whether other is a ClericResources with the same uses and state.
func (r ClericResources) Equal(other FeatureState) bool {
	o, ok := other.(ClericResources)
	return ok && o.channelMax == r.channelMax && o.channelRemaining == r.channelRemaining && o.DivineIntervention() == r.DivineIntervention()
}

// AfterRest returns the resources after the benefits of a rest of kind (SRD 5.2.1,
// "Level 2: Channel Divinity", "Level 10: Divine Intervention", "Level 20: Greater
// Divine Intervention", pp. 37-38): a Short Rest regains one expended Channel Divinity
// use; a Long Rest regains all of them, makes a used Divine Intervention available and
// counts a recharging one down by one Long Rest. Any other kind changes nothing. The
// host passes RestApplied.BenefitsOf.
func (r ClericResources) AfterRest(kind RestKind) ClericResources {
	r.intervention = r.DivineIntervention()
	switch kind {
	case ShortRest:
		r.channelRemaining = min(r.channelRemaining+1, r.channelMax)
	case LongRest:
		r.channelRemaining = r.channelMax
		switch s := r.intervention.(type) {
		case DivineInterventionUnavailableUntilLongRest:
			r.intervention = DivineInterventionAvailable{}
		case DivineInterventionRecharging:
			r.intervention = DivineInterventionAvailable{}
			if s.LongRestsRemaining > 1 {
				r.intervention = DivineInterventionRecharging{LongRestsRemaining: s.LongRestsRemaining - 1}
			}
		}
	}
	return r
}

// readClericChoices returns the ClericChoices c stores, or the empty choices when none
// is stored. It returns an error matching srderr.ErrInvalidInput when another type of
// state sits under the key.
func readClericChoices(c Character) (ClericChoices, error) {
	s, ok, err := ReadFeatureState(c.FeatureStates(), ClericChoicesKey())
	if err != nil || !ok {
		return ClericChoices{}, err
	}
	return s, nil
}

// readClericResources returns the ClericResources c stores, or InitialClericResources
// of initialMax when none is stored. It returns an error matching
// srderr.ErrInvalidInput when another type of state sits under the key.
func readClericResources(c Character, initialMax int) (ClericResources, error) {
	s, ok, err := ReadFeatureState(c.FeatureStates(), ClericResourcesKey())
	if err != nil {
		return ClericResources{}, err
	}
	if !ok {
		return InitialClericResources(initialMax), nil
	}
	return s, nil
}
