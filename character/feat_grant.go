package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// FeatSource is where a feat comes from, a closed set (SRD 5.2.1, pp. 24, 47-49,
// 54, 59, 73, 83, 86-87). Every variant is comparable, so two sources are the
// same when they are ==. Variants: BackgroundFeat, CustomBackgroundFeat,
// SpeciesTraitFeat, ClassFeatureFeat, EldritchInvocationFeat, BonusFeat.
type FeatSource interface{ isFeatSource() }

// BackgroundFeat is the feat a background grants. Construct it with a keyed
// literal.
type BackgroundFeat struct{ Background content.BackgroundID }

// CustomBackgroundFeat is the feat a custom background grants (SRD 5.2.1,
// "Creating a Background", p. 193).
type CustomBackgroundFeat struct{}

// SpeciesTraitFeat is the feat a species trait grants, such as Human Versatile.
// Construct it with a keyed literal.
type SpeciesTraitFeat struct {
	Species content.SpeciesID
	Trait   content.FeatureID
}

// ClassFeatureFeat is the feat a class feature grants, such as Fighting Style or
// Ability Score Improvement. Construct it with a keyed literal.
type ClassFeatureFeat struct {
	Class   content.ClassID
	Feature content.FeatureID
}

// EldritchInvocationFeat is the feat an Eldritch Invocation grants (SRD 5.2.1,
// "Lessons of the First Ones", p. 73). Construct it with a keyed literal.
type EldritchInvocationFeat struct{ Option content.FeatureOptionID }

// BonusFeat is a feat earned with the Bonus Feats at Level 20 option (SRD
// 5.2.1, p. 24).
type BonusFeat struct{}

func (BackgroundFeat) isFeatSource()         {}
func (CustomBackgroundFeat) isFeatSource()   {}
func (SpeciesTraitFeat) isFeatSource()       {}
func (ClassFeatureFeat) isFeatSource()       {}
func (EldritchInvocationFeat) isFeatSource() {}
func (BonusFeat) isFeatSource()              {}

func validFeatSource(s FeatSource) bool {
	switch v := s.(type) {
	case BackgroundFeat:
		return v.Background.Valid()
	case CustomBackgroundFeat, BonusFeat:
		return true
	case SpeciesTraitFeat:
		return v.Species.Valid() && v.Trait.Valid()
	case ClassFeatureFeat:
		return v.Class.Valid() && v.Feature.Valid()
	case EldritchInvocationFeat:
		return v.Option.Valid()
	}
	return false
}

// RequiredSpellList is the spell list a granted Magic Initiate must use, a closed
// set. Variants: NoRequiredSpellList, RequiredSpellListIs.
type RequiredSpellList interface{ isRequiredSpellList() }

// NoRequiredSpellList places no constraint on the spell list.
type NoRequiredSpellList struct{}

// RequiredSpellListIs requires the spell list of Class (a background's printed
// "Magic Initiate (Cleric)"). Construct it with a keyed literal.
type RequiredSpellListIs struct{ Class content.ClassID }

func (NoRequiredSpellList) isRequiredSpellList() {}
func (RequiredSpellListIs) isRequiredSpellList() {}

// FeatGrantScope is which feats a grant allows, a closed set (SRD 5.2.1,
// "Parts of a Feat", Category, p. 87). Variants: SpecifiedFeat,
// FeatFromCategory, AnyFeat.
type FeatGrantScope interface{ isFeatGrantScope() }

// SpecifiedFeat allows only Feat, with the spell list Required. Construct it
// with a keyed literal.
type SpecifiedFeat struct {
	Feat     content.FeatID
	Required RequiredSpellList
}

// FeatFromCategory allows a feat of Category. Construct it with a keyed literal.
type FeatFromCategory struct{ Category content.FeatCategory }

// AnyFeat allows a feat of any of the four categories.
type AnyFeat struct{}

func (SpecifiedFeat) isFeatGrantScope()    {}
func (FeatFromCategory) isFeatGrantScope() {}
func (AnyFeat) isFeatGrantScope()          {}

// FeatGrant is a source and the scope of the feats it allows. It is comparable.
// The zero FeatGrant is invalid; see IsZero.
type FeatGrant struct {
	source FeatSource
	scope  FeatGrantScope
}

// NewFeatGrant returns the grant of scope from source. It returns an error
// matching srderr.ErrInvalidInput for a nil or invalid source, scope, feat id,
// category or required list. It never panics.
func NewFeatGrant(source FeatSource, scope FeatGrantScope) (FeatGrant, error) {
	ok := validFeatSource(source)
	switch s := scope.(type) {
	case SpecifiedFeat:
		switch r := s.Required.(type) {
		case NoRequiredSpellList:
		case RequiredSpellListIs:
			ok = ok && r.Class.Valid()
		default:
			ok = false
		}
		ok = ok && s.Feat.Valid()
	case FeatFromCategory:
		ok = ok && s.Category.Valid()
	case AnyFeat:
	default:
		ok = false
	}
	if !ok {
		return FeatGrant{}, invalidInput("feat grant")
	}
	return FeatGrant{source: source, scope: scope}, nil
}

// Source returns where the feat comes from; nil for the zero FeatGrant.
func (g FeatGrant) Source() FeatSource { return g.source }

// Scope returns which feats the grant allows; nil for the zero FeatGrant.
func (g FeatGrant) Scope() FeatGrantScope { return g.scope }

// IsZero reports whether g is the invalid zero FeatGrant.
func (g FeatGrant) IsZero() bool { return g.source == nil }

// allows reports whether the grant's scope allows feat.
func (g FeatGrant) allows(feat content.FeatID, category content.FeatCategory) bool {
	switch s := g.scope.(type) {
	case SpecifiedFeat:
		return s.Feat == feat
	case FeatFromCategory:
		return s.Category == category
	}
	return true
}

// maxHeldFeatures bounds the features a HeldFeatures holds.
const maxHeldFeatures = 1024

// HeldFeatures is the set of class features a character holds, which the
// feature prerequisites of feats read (SRD 5.2.1, "Parts of a Feat",
// Prerequisite, p. 87). The host builds it from the class specs. It is immutable
// and copied on construction and on read; compare two with Equal. The zero value
// is the empty set. Safe for concurrent use.
type HeldFeatures struct{ ids []content.FeatureID }

// NewHeldFeatures returns the set of ids, sorted without repeats.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid id and for
// more than 1024 ids. It never panics.
func NewHeldFeatures(ids ...content.FeatureID) (HeldFeatures, error) {
	if len(ids) > maxHeldFeatures || slices.ContainsFunc(ids, func(id content.FeatureID) bool { return !id.Valid() }) {
		return HeldFeatures{}, invalidInput("held features")
	}
	out := slices.Clone(ids)
	slices.Sort(out)
	return HeldFeatures{ids: slices.Compact(out)}, nil
}

// Contains reports whether id is held.
func (h HeldFeatures) Contains(id content.FeatureID) bool {
	_, ok := slices.BinarySearch(h.ids, id)
	return ok
}

// IDs returns a new slice of the held features, sorted.
func (h HeldFeatures) IDs() []content.FeatureID { return slices.Clone(h.ids) }

// Equal reports whether h and o hold the same features.
func (h HeldFeatures) Equal(o HeldFeatures) bool { return slices.Equal(h.ids, o.ids) }
