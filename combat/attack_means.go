package combat

import "github.com/Whealinghold/srd5e-go/content"

// AttackMeans is the kind of attack the Attack action makes (SRD 5.2.1, "Attack
// [Action]", p. 177: "one attack roll with a weapon or an Unarmed Strike"). It is
// bookkeeping, not an AttackForm. PactWeapon exists because Thirsting Blade's extra
// attack is "for your pact weapon only" (p. 74). No value stands for a spell attack.
// The zero value is not a means. Comparable and safe for concurrent use.
type AttackMeans int

// The three means of attack.
const (
	AttackMeansPactWeapon AttackMeans = iota + 1
	AttackMeansOtherWeapon
	AttackMeansUnarmedStrike
)

// Valid reports whether m is one of the three means.
func (m AttackMeans) Valid() bool { return m >= AttackMeansPactWeapon && m <= AttackMeansUnarmedStrike }

// String returns "pact_weapon", "other_weapon", "unarmed_strike" or "unknown".
func (m AttackMeans) String() string {
	return enumName([]string{"pact_weapon", "other_weapon", "unarmed_strike"}, int(m))
}

// AttackReplacementKind says whether an attack was replaced by another effect or
// forgone (SRD 5.2.1, "Breath Weapon", p. 84; "Pact of the Chain", p. 74). The zero
// value is not a kind.
type AttackReplacementKind int

// The two kinds of replacement.
const (
	AttackReplacementKindReplace AttackReplacementKind = iota + 1
	AttackReplacementKindForgo
)

// Valid reports whether k is one of the two kinds.
func (k AttackReplacementKind) Valid() bool {
	return k >= AttackReplacementKindReplace && k <= AttackReplacementKindForgo
}

// String returns "replace", "forgo" or "unknown".
func (k AttackReplacementKind) String() string { return enumName([]string{"replace", "forgo"}, int(k)) }

// AttackReplacementSource names what replaced or forwent an attack: a feature or an
// item. It is a closed set built with ReplacementByFeature or ReplacementByItem. The
// zero value is invalid. Comparable and safe for concurrent use.
type AttackReplacementSource struct {
	feature content.FeatureID
	item    content.ItemID
}

// ReplacementByFeature returns the source that is the feature (a species trait is a
// feature, so the Breath Weapon trait is one). It is invalid when id is invalid.
func ReplacementByFeature(id content.FeatureID) AttackReplacementSource {
	return AttackReplacementSource{feature: id}
}

// ReplacementByItem returns the source that is the item (thrown Acid, a Net and so
// on). It is invalid when id is invalid.
func ReplacementByItem(id content.ItemID) AttackReplacementSource {
	return AttackReplacementSource{item: id}
}

// Valid reports whether s names exactly one valid feature or item.
func (s AttackReplacementSource) Valid() bool {
	return (s.feature.Valid() && s.item == "") || (s.item.Valid() && s.feature == "")
}

// Feature returns the feature and true when s is a feature source.
func (s AttackReplacementSource) Feature() (content.FeatureID, bool) {
	return s.feature, s.feature != "" && s.Valid()
}

// Item returns the item and true when s is an item source.
func (s AttackReplacementSource) Item() (content.ItemID, bool) {
	return s.item, s.item != "" && s.Valid()
}

// AttackReplacement is one attack of the Attack action that was replaced by an effect
// or forgone ("replace one of your attacks", "forgo one of your own attacks"). Build
// it with NewAttackReplacement. The zero value is invalid. Comparable and safe for
// concurrent use.
type AttackReplacement struct {
	kind   AttackReplacementKind
	source AttackReplacementSource
}

// NewAttackReplacement returns the replacement. It returns an error matching
// srderr.ErrInvalidInput for an invalid kind or source. It never panics.
func NewAttackReplacement(kind AttackReplacementKind, source AttackReplacementSource) (AttackReplacement, error) {
	if !kind.Valid() || !source.Valid() {
		return AttackReplacement{}, invalidInput("attack replacement needs a kind and a source")
	}
	return AttackReplacement{kind, source}, nil
}

// Kind returns whether the attack was replaced or forgone.
func (r AttackReplacement) Kind() AttackReplacementKind { return r.kind }

// Source returns what replaced or forwent the attack.
func (r AttackReplacement) Source() AttackReplacementSource { return r.source }

// Valid reports whether r was built by NewAttackReplacement.
func (r AttackReplacement) Valid() bool { return r.kind.Valid() && r.source.Valid() }

// AttackEntry is one attack counted against the allowance of an Attack action: an
// attack made with a means, or a replacement. Build it with MadeAttack or
// ReplacedAttack. The zero value is invalid. Comparable and safe for concurrent use.
type AttackEntry struct {
	means       AttackMeans
	replacement AttackReplacement
}

// MadeAttack returns the entry for an attack made with the means. It is invalid when
// means is invalid.
func MadeAttack(means AttackMeans) AttackEntry { return AttackEntry{means: means} }

// ReplacedAttack returns the entry for a replaced or forgone attack. It is invalid
// when r is invalid.
func ReplacedAttack(r AttackReplacement) AttackEntry { return AttackEntry{replacement: r} }

// Valid reports whether e is exactly one valid means or one valid replacement.
func (e AttackEntry) Valid() bool {
	return (e.means.Valid() && e.replacement == AttackReplacement{}) || (e.replacement.Valid() && e.means == 0)
}

// IsReplacement reports whether e is a replaced or forgone attack.
func (e AttackEntry) IsReplacement() bool { return e.replacement != AttackReplacement{} }

// Means returns the means of an attack made, and the zero AttackMeans for a replacement.
func (e AttackEntry) Means() AttackMeans { return e.means }

// Replacement returns the replacement, and the invalid zero one for an attack made.
func (e AttackEntry) Replacement() AttackReplacement { return e.replacement }

// pactCompatible reports whether e can be part of a pact weapon-only action: a pact
// weapon attack or a replacement (SRD 5.2.1, "Thirsting Blade", p. 74).
func (e AttackEntry) pactCompatible() bool {
	return e.IsReplacement() || e.means == AttackMeansPactWeapon
}
