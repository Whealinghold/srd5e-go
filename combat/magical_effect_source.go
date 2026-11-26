package combat

import "github.com/Whealinghold/srd5e-go/content"

// EffectSourceKind says where a Magical Effect or other effect comes from: a
// spell, a magic item, a phenomenon a rule labels as magical, or something a
// rule does not label as magical. The zero value is not a kind.
type EffectSourceKind int

// The four kinds of effect source.
const (
	// EffectSourceSpell: a spell, which is a magical effect (SRD 5.2.1, "Spell", p. 188).
	EffectSourceSpell EffectSourceKind = iota + 1
	// EffectSourceMagicItem: a magic item.
	EffectSourceMagicItem
	// EffectSourceLabeledMagical: a phenomenon that a rule labels as magical.
	EffectSourceLabeledMagical
	// EffectSourceNonMagical: a phenomenon no rule labels as magical.
	EffectSourceNonMagical
)

// String returns "spell", "magic_item", "labeled_magical", "non_magical" or "unknown".
func (k EffectSourceKind) String() string {
	return enumName([]string{"spell", "magic_item", "labeled_magical", "non_magical"}, int(k))
}

// EffectSource is where an effect comes from (SRD 5.2.1, "Magical Effect", p. 185:
// "created by a spell, a magic item, or a phenomenon that a rule labels as
// magical"). Build one with SpellSource, MagicItemSource, LabeledMagicalSource or
// NonMagicalSource. It is a value: two sources are equal exactly when their kind
// and id are equal, so a spell and a magic item with the same id text differ. The
// zero value is invalid. Comparable and safe for concurrent use.
type EffectSource struct {
	kind EffectSourceKind
	id   string
}

func newEffectSource(kind EffectSourceKind, id string, valid bool) (EffectSource, error) {
	if !valid {
		return EffectSource{}, invalidInput("effect source needs a valid content id")
	}
	return EffectSource{kind, id}, nil
}

// SpellSource returns the source for a spell. It returns an error matching
// srderr.ErrInvalidInput when id is not a valid id. It never panics.
func SpellSource(id content.SpellID) (EffectSource, error) {
	return newEffectSource(EffectSourceSpell, id.String(), id.Valid())
}

// MagicItemSource returns the source for a magic item. It returns an error
// matching srderr.ErrInvalidInput when id is not a valid id. It never panics.
func MagicItemSource(id content.MagicItemID) (EffectSource, error) {
	return newEffectSource(EffectSourceMagicItem, id.String(), id.Valid())
}

// LabeledMagicalSource returns the source for an effect "a rule labels as
// magical" (SRD 5.2.1, "Magical Effect", p. 185), named by its feature id. It
// returns an error matching srderr.ErrInvalidInput when id is not a valid id. It
// never panics.
func LabeledMagicalSource(id content.FeatureID) (EffectSource, error) {
	return newEffectSource(EffectSourceLabeledMagical, id.String(), id.Valid())
}

// NonMagicalSource returns the source for an effect that is not a Magical Effect,
// named by its feature id. It returns an error matching srderr.ErrInvalidInput
// when id is not a valid id. It never panics.
func NonMagicalSource(id content.FeatureID) (EffectSource, error) {
	return newEffectSource(EffectSourceNonMagical, id.String(), id.Valid())
}

// Kind returns the kind of the source, 0 for the zero value.
func (s EffectSource) Kind() EffectSourceKind { return s.kind }

// ID returns the content id text of the source, empty for the zero value.
func (s EffectSource) ID() string { return s.id }

// IsZero reports whether s is the invalid zero value.
func (s EffectSource) IsZero() bool { return s.kind == 0 }

// IsMagical reports whether the source is a Magical Effect: a spell, a magic item
// or a phenomenon a rule labels as magical (SRD 5.2.1, "Magical Effect", p. 185).
func (s EffectSource) IsMagical() bool {
	return s.kind == EffectSourceSpell || s.kind == EffectSourceMagicItem || s.kind == EffectSourceLabeledMagical
}
