package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// MulticlassTraining is the domain service for the proficiencies a first level
// in a new class grants (SRD 5.2.1, "Multiclassing", "Proficiencies", p. 25).
// The zero value is ready to use. It is stateless and safe for concurrent use.
type MulticlassTraining struct{}

// GrantsFor returns, for each class of levels after the first (the initial
// class), that class's MulticlassGrants in ClassLevels order, and an empty
// slice for a single-class character. The result is the value a host passes to
// derived stats as multiclass armor training and uses for weapon, tool and
// skill training.
//
// It returns an error matching srderr.ErrInvalidInput for empty levels or when
// a class after the first has no definition in definitions. It never panics.
func (MulticlassTraining) GrantsFor(levels ClassLevels, definitions map[content.ClassID]content.ClassDefinition) ([]content.MulticlassGrants, error) {
	if levels.IsZero() {
		return nil, invalidInput("multiclass grants of empty class levels")
	}
	out := []content.MulticlassGrants{}
	for _, e := range levels.entries[1:] {
		def, ok := definitions[e.Class]
		if !ok {
			return nil, invalidInput("no definition for class " + e.Class.String())
		}
		out = append(out, def.MulticlassGrants())
	}
	return out, nil
}
