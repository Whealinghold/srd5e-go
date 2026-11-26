package combat

import (
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
)

// Condition is one of the fifteen conditions of the Rules Glossary (SRD 5.2.1,
// "Condition", p. 179). It is a closed set with one value type per
// content.ConditionKind; Kind says which. Two Conditions are equal under ==
// when they have the same kind and the same source or level. Variants:
// Blinded, Charmed, Deafened, ExhaustionCondition, Frightened, Grappled, Incapacitated,
// Invisible, Paralyzed, Petrified, Poisoned, Prone, Restrained, Stunned,
// Unconscious. A nil Condition and a variant whose source or level is the zero
// value are invalid; NewConditionSet and the commands reject them.
type Condition interface {
	condition()
	Kind() content.ConditionKind
}

// Blinded is the Blinded condition (p. 177).
type Blinded struct{}

// Charmed is the Charmed condition (p. 178). Charmer says who charmed the
// creature; the zero value holds no Charmer and is invalid. Construct with a
// keyed literal.
type Charmed struct{ Charmer Charmer }

// Deafened is the Deafened condition (p. 181).
type Deafened struct{}

// ExhaustionCondition is the Exhaustion condition at Level (p. 181). The zero value holds
// the invalid zero level. Construct with a keyed literal.
type ExhaustionCondition struct{ Level ExhaustionLevel }

// Frightened is the Frightened condition (p. 182). Source is the creature that
// causes the fear; the zero value holds no source and is invalid. Construct with
// a keyed literal.
type Frightened struct{ Source CombatantID }

// Grappled is the Grappled condition (p. 182). Grappler is the creature holding
// this one; the zero value holds no grappler and is invalid. Construct with a
// keyed literal.
type Grappled struct{ Grappler CombatantID }

// Incapacitated is the Incapacitated condition (p. 184).
type Incapacitated struct{}

// Invisible is the Invisible condition (p. 184).
type Invisible struct{}

// Paralyzed is the Paralyzed condition (p. 186).
type Paralyzed struct{}

// Petrified is the Petrified condition (p. 186).
type Petrified struct{}

// Poisoned is the Poisoned condition (p. 186).
type Poisoned struct{}

// Prone is the Prone condition (p. 186).
type Prone struct{}

// Restrained is the Restrained condition (p. 187).
type Restrained struct{}

// Stunned is the Stunned condition (p. 189).
type Stunned struct{}

// Unconscious is the Unconscious condition (p. 191).
type Unconscious struct{}

func (Blinded) condition()             {}
func (Charmed) condition()             {}
func (Deafened) condition()            {}
func (ExhaustionCondition) condition() {}
func (Frightened) condition()          {}
func (Grappled) condition()            {}
func (Incapacitated) condition()       {}
func (Invisible) condition()           {}
func (Paralyzed) condition()           {}
func (Petrified) condition()           {}
func (Poisoned) condition()            {}
func (Prone) condition()               {}
func (Restrained) condition()          {}
func (Stunned) condition()             {}
func (Unconscious) condition()         {}

// Kind returns content.ConditionBlinded.
func (Blinded) Kind() content.ConditionKind { return content.ConditionBlinded }

// Kind returns content.ConditionCharmed.
func (Charmed) Kind() content.ConditionKind { return content.ConditionCharmed }

// Kind returns content.ConditionDeafened.
func (Deafened) Kind() content.ConditionKind { return content.ConditionDeafened }

// Kind returns content.ConditionExhaustion.
func (ExhaustionCondition) Kind() content.ConditionKind { return content.ConditionExhaustion }

// Kind returns content.ConditionFrightened.
func (Frightened) Kind() content.ConditionKind { return content.ConditionFrightened }

// Kind returns content.ConditionGrappled.
func (Grappled) Kind() content.ConditionKind { return content.ConditionGrappled }

// Kind returns content.ConditionIncapacitated.
func (Incapacitated) Kind() content.ConditionKind { return content.ConditionIncapacitated }

// Kind returns content.ConditionInvisible.
func (Invisible) Kind() content.ConditionKind { return content.ConditionInvisible }

// Kind returns content.ConditionParalyzed.
func (Paralyzed) Kind() content.ConditionKind { return content.ConditionParalyzed }

// Kind returns content.ConditionPetrified.
func (Petrified) Kind() content.ConditionKind { return content.ConditionPetrified }

// Kind returns content.ConditionPoisoned.
func (Poisoned) Kind() content.ConditionKind { return content.ConditionPoisoned }

// Kind returns content.ConditionProne.
func (Prone) Kind() content.ConditionKind { return content.ConditionProne }

// Kind returns content.ConditionRestrained.
func (Restrained) Kind() content.ConditionKind { return content.ConditionRestrained }

// Kind returns content.ConditionStunned.
func (Stunned) Kind() content.ConditionKind { return content.ConditionStunned }

// Kind returns content.ConditionUnconscious.
func (Unconscious) Kind() content.ConditionKind { return content.ConditionUnconscious }

// Charmer says who or what charmed a creature (SRD 5.2.1, "Charmed", p. 178;
// "Conflict", p. 208). Variants: CharmedByCombatant, CharmedByItem. Both are
// comparable with ==.
type Charmer interface{ charmer() }

// CharmedByCombatant is a charmer that is a creature in the fight. The zero
// value holds the invalid zero CombatantID. Construct with a keyed literal.
type CharmedByCombatant struct{ ID CombatantID }

// CharmedByItem is a charmer that is an item, such as a sentient item whose
// control effect gives the Charmed condition. The id is an opaque non-blank
// string the host supplies and this package never interprets. Build it with
// NewCharmedByItem; the zero value is invalid.
type CharmedByItem struct{ itemID string }

func (CharmedByCombatant) charmer() {}
func (CharmedByItem) charmer()      {}

// NewCharmedByItem returns the charmer for itemID. It returns an error matching
// srderr.ErrInvalidInput when itemID is empty or only whitespace. It never
// panics.
func NewCharmedByItem(itemID string) (CharmedByItem, error) {
	if strings.TrimSpace(itemID) == "" {
		return CharmedByItem{}, invalidInput("charmer item id must not be blank")
	}
	return CharmedByItem{itemID}, nil
}

// ItemID returns the opaque item id.
func (c CharmedByItem) ItemID() string { return c.itemID }

// validCharmer reports whether c is a usable Charmer.
func validCharmer(c Charmer) bool {
	switch v := c.(type) {
	case CharmedByCombatant:
		return !v.ID.IsZero()
	case CharmedByItem:
		return v.itemID != ""
	}
	return false
}

// validCondition reports whether c is one of the fifteen value types with its
// source or level set.
func validCondition(c Condition) bool {
	switch v := c.(type) {
	case Charmed:
		return validCharmer(v.Charmer)
	case Frightened:
		return !v.Source.IsZero()
	case Grappled:
		return !v.Grappler.IsZero()
	case ExhaustionCondition:
		return !v.Level.IsZero()
	case Blinded, Deafened, Incapacitated, Invisible, Paralyzed, Petrified, Poisoned,
		Prone, Restrained, Stunned, Unconscious:
		return true
	}
	return false
}
