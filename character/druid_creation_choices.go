package character

import "github.com/Whealinghold/srd5e-go/content"

// DruidCreationChoices is the ClassCreationChoices of a new Druid: the Primal Order chosen at level 1 (class-druid.md FR-163,
// FR-170). It seeds a DruidFeatureState holding the order and nothing else under DruidFeatureStateKey, and names no content.
// The two skills and the Starting Equipment option are validated by CreateCharacter against the class data, and a multiclass
// Druid's training is level advancement's. Build it with NewDruidCreationChoices; the zero value seeds no order. Immutable,
// comparable with ==. Safe for concurrent use.
type DruidCreationChoices struct{ order PrimalOrder }

// NewDruidCreationChoices returns the choices with the Primal Order. It returns an error matching srderr.ErrInvalidInput for an
// order outside Magician and Warden. It never panics.
func NewDruidCreationChoices(order PrimalOrder) (DruidCreationChoices, error) {
	if !order.Valid() {
		return DruidCreationChoices{}, invalidInput("Primal Order outside the closed set")
	}
	return DruidCreationChoices{order: order}, nil
}

// PrimalOrder returns the order chosen.
func (c DruidCreationChoices) PrimalOrder() PrimalOrder { return c.order }

// SeedFeatureStates returns states with the DruidFeatureState of a level 1 Druid added and every other entry unchanged; states
// itself is unchanged. States that already hold the key are returned unchanged: CreateCharacter starts from empty states, so
// that is a programmer error, and the package does not panic outside Must helpers.
func (c DruidCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	if _, held, err := ReadFeatureState(states, DruidFeatureStateKey()); held || err != nil {
		return states
	}
	next, err := SetFeatureState(states, DruidFeatureStateKey(), DruidFeatureState{order: c.order})
	if err != nil { // unreachable: the key is valid, the state is non-nil and the name is not held
		return states
	}
	return next
}

// ContentKeys returns an empty slice: the Primal Order names no content entry.
func (DruidCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

var _ ClassCreationChoices = DruidCreationChoices{}
