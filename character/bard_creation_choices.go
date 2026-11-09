package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// BardCreationChoices is the ClassCreationChoices of a new Bard: it seeds a full Bardic Inspiration pool under
// BardicInspirationPoolKey, sized from the Charisma modifier it carries because the seeding method receives no
// ability scores (class-bard.md FR-117, FR-121). The skill, Musical Instrument and Starting Equipment choices of a
// first level are checked by CreateCharacter against the class data. It names no content. It is immutable and
// comparable with ==. Safe for concurrent use.
type BardCreationChoices struct{ charisma ability.Modifier }

// NewBardCreationChoices returns the choices of a Bard whose Charisma modifier is charisma.
func NewBardCreationChoices(charisma ability.Modifier) BardCreationChoices {
	return BardCreationChoices{charisma: charisma}
}

// SeedFeatureStates returns states with a full Bardic Inspiration pool added and every other entry unchanged;
// states itself is unchanged. States that already hold the key are returned unchanged: CreateCharacter starts from
// empty states, so that is a programmer error, and the package does not panic outside Must helpers.
func (c BardCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	return seedBardicInspirationPool(states, c.charisma)
}

// ContentKeys returns an empty slice: the first-level choices name no content the choices themselves load.
func (BardCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

func seedBardicInspirationPool(states FeatureStates, charisma ability.Modifier) FeatureStates {
	if _, held, err := ReadFeatureState(states, BardicInspirationPoolKey()); held || err != nil {
		return states
	}
	next, err := SetFeatureState(states, BardicInspirationPoolKey(), InitialBardicInspirationPool(charisma))
	if err != nil { // unreachable: the key is valid, the states are non-nil and the name is not held
		return states
	}
	return next
}

// BardMulticlassChoicesError is the error NewBardMulticlassChoices returns. It lists every problem together
// (class-bard.md FR-015, FR-110) and matches srderr.ErrInvalidInput. Build it with a keyed literal.
type BardMulticlassChoicesError struct{ Problems []BardChoiceProblem }

// Error renders the error. The text is not a contract.
func (e *BardMulticlassChoicesError) Error() string {
	out := "character: invalid Bard multiclass choices:"
	for _, p := range e.Problems {
		out += " " + p.String()
	}
	return out
}

// Unwrap returns srderr.ErrInvalidInput.
func (*BardMulticlassChoicesError) Unwrap() error { return srderr.ErrInvalidInput }

// BardMulticlassChoices is the ClassCreationChoices of a character taking its first Bard level: the one skill and the
// one Musical Instrument the Bard gives a multiclass character (SRD 5.2.1, "As a Multiclass Character", p. 31), and
// the Charisma modifier that sizes the pool. Build it with NewBardMulticlassChoices. Immutable. Safe for concurrent use.
type BardMulticlassChoices struct {
	skills     []content.Skill
	instrument content.ItemID
	charisma   ability.Modifier
}

// NewBardMulticlassChoices validates a multiclass entry: exactly one valid skill the character is not already proficient
// in (FR-110), exactly one instrument, and that instrument in instruments, the caller's set of Musical Instrument
// items (FR-122: an item is a Musical Instrument exactly when its id is in the set, and the function never reads a
// content source). instrument is the empty id when none was named. It returns a *BardMulticlassChoicesError, matching
// srderr.ErrInvalidInput, listing every problem. It never panics.
func NewBardMulticlassChoices(skills []content.Skill, instrument []content.ItemID, charisma ability.Modifier, held []content.Skill, instruments map[content.ItemID]bool) (BardMulticlassChoices, error) {
	var problems []BardChoiceProblem
	if len(skills) != 1 {
		problems = append(problems, BardSkillCountWrong)
	}
	if slices.ContainsFunc(skills, func(s content.Skill) bool { return slices.Contains(held, s) }) {
		problems = append(problems, BardSkillAlreadyProficient)
	}
	if slices.ContainsFunc(skills, func(s content.Skill) bool { return !s.Valid() }) {
		problems = append(problems, BardSkillInvalid)
	}
	if len(instrument) != 1 {
		problems = append(problems, BardInstrumentCountWrong)
	} else if !instruments[instrument[0]] {
		problems = append(problems, BardInstrumentNotMusical)
	}
	if len(problems) != 0 {
		return BardMulticlassChoices{}, &BardMulticlassChoicesError{Problems: problems}
	}
	return BardMulticlassChoices{skills: slices.Clone(skills), instrument: instrument[0], charisma: charisma}, nil
}

// Skills returns the skill chosen, as a new slice.
func (c BardMulticlassChoices) Skills() []content.Skill { return slices.Clone(c.skills) }

// Instrument returns the Musical Instrument chosen.
func (c BardMulticlassChoices) Instrument() content.ItemID { return c.instrument }

// SeedFeatureStates returns states with a full Bardic Inspiration pool added (see BardCreationChoices.SeedFeatureStates).
func (c BardMulticlassChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	return seedBardicInspirationPool(states, c.charisma)
}

// ContentKeys returns an empty slice: items are not a content kind yet, so the instrument is named by its id alone.
func (BardMulticlassChoices) ContentKeys() []content.Key { return []content.Key{} }

var (
	_ ClassCreationChoices = BardCreationChoices{}
	_ ClassCreationChoices = BardMulticlassChoices{}
)
