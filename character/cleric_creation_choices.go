package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ClericCreationChoicesError is the error NewClericCreationChoices returns for a Divine Order
// choice the rules refuse (class-cleric.md FR-140). It matches srderr.ErrInvalidInput with
// errors.Is and another *ClericCreationChoicesError of the same Reason. Build it with a
// keyed literal.
type ClericCreationChoicesError struct{ Reason ClericReason }

// Error renders the error. The text is not a contract.
func (e *ClericCreationChoicesError) Error() string {
	return "character: invalid Cleric creation choices: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (*ClericCreationChoicesError) Unwrap() error { return srderr.ErrInvalidInput }

// Is matches another *ClericCreationChoicesError of the same Reason.
func (e *ClericCreationChoicesError) Is(target error) bool {
	t, ok := target.(*ClericCreationChoicesError)
	return ok && t.Reason == e.Reason
}

// ClericCreationChoices is the ClassCreationChoices of a new Cleric: it seeds the validated Divine
// Order under ClericChoicesKey and the level 1 resources (Channel Divinity 0 of 0, Divine
// Intervention available) under ClericResourcesKey, and names the Thaumaturge's extra cantrip as
// the content it refers to. Build it with NewClericCreationChoices; the zero value seeds an empty
// order. It is immutable and comparable with ==. Safe for concurrent use.
type ClericCreationChoices struct {
	order        DivineOrder
	extraCantrip content.SpellID
}

// NewClericCreationChoices applies the ChooseDivineOrder rules (without already_chosen) to the
// requested order names, the Thaumaturge's extra cantrip and the cantrips already known. It
// returns a *ClericCreationChoicesError, matching srderr.ErrInvalidInput, carrying the first
// reason that holds. It never panics.
func NewClericCreationChoices(names []string, extra *content.Spell, known []content.SpellID) (ClericCreationChoices, error) {
	order, reason := divineOrderProblem(names, extra, known)
	if reason != 0 {
		return ClericCreationChoices{}, &ClericCreationChoicesError{Reason: reason}
	}
	out := ClericCreationChoices{order: order}
	if order == DivineOrderThaumaturge {
		out.extraCantrip = extra.ID()
	}
	return out, nil
}

// SeedFeatureStates returns states with the ClericChoices and the initial ClericResources added
// and every other entry unchanged; states itself is unchanged. States that already hold either key are returned unchanged:
// CreateCharacter starts from empty states, so that is a programmer error, and the package does not panic outside Must helpers.
func (c ClericCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	_, choicesHeld, err1 := ReadFeatureState(states, ClericChoicesKey())
	_, resourcesHeld, err2 := ReadFeatureState(states, ClericResourcesKey())
	if choicesHeld || resourcesHeld || err1 != nil || err2 != nil {
		return states
	}
	next, err := SetFeatureState(states, ClericChoicesKey(), ClericChoices{order: c.order, extraCantrip: c.extraCantrip})
	if err == nil {
		next, err = SetFeatureState(next, ClericResourcesKey(), InitialClericResources(0))
	}
	if err != nil { // unreachable: both keys are valid, the states are non-nil and neither name is held
		return states
	}
	return next
}

// ContentKeys returns the key of the Thaumaturge's extra cantrip and none for a Protector, as a new slice.
func (c ClericCreationChoices) ContentKeys() []content.Key {
	if c.extraCantrip == "" {
		return []content.Key{}
	}
	return []content.Key{content.NewKey(content.KindSpell, content.MustID(string(c.extraCantrip)))}
}

var _ ClassCreationChoices = ClericCreationChoices{}
