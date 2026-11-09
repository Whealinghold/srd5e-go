package character

import "github.com/Whealinghold/srd5e-go/content"

// ClassCreationChoices is what a class asks of a new character beyond the
// shared creation data (for example a choice of feature): it seeds the feature
// states and names the content entries the choices refer to, so the creation
// use case can look them up. Implementations are pure, immutable and safe for
// concurrent use; the aggregate stores nothing class-specific.
type ClassCreationChoices interface {
	// SeedFeatureStates returns the feature states for the new character: states
	// with the class's entries added. It MUST NOT alter its argument.
	SeedFeatureStates(states FeatureStates) FeatureStates
	// ContentKeys returns a new slice of the content keys the choices name.
	ContentKeys() []content.Key
}

// NoClassCreationChoices is the default ClassCreationChoices: it seeds nothing
// and names no content.
type NoClassCreationChoices struct{}

// SeedFeatureStates returns states unchanged.
func (NoClassCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates { return states }

// ContentKeys returns no keys.
func (NoClassCreationChoices) ContentKeys() []content.Key { return nil }
