package content

import "slices"

// FeatBenefit is one benefit of a feat: an optional printed name and its
// verbatim text (SRD 5.2.1, "Parts of a Feat", p. 87). The zero value is
// invalid; build it with [NewFeatBenefit]. Comparable.
type FeatBenefit struct {
	name string
	text string
}

// NewFeatBenefit returns a benefit with the verbatim text, called name, or
// unnamed when name is "".
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidName for a non-empty name that breaks the name rule (schema
// FR-027), then BlockBlankText or BlockInvalidCharacter for a text that breaks
// the text rule (schema FR-028). It never panics.
func NewFeatBenefit(name, text string) (FeatBenefit, error) {
	if name != "" { // equivalent mutant: a non-empty string is above "", so != and > agree
		if err := checkName("FeatBenefit", name); err != nil {
			return FeatBenefit{}, err
		}
	}
	if err := checkText("FeatBenefit", text); err != nil {
		return FeatBenefit{}, err
	}
	return FeatBenefit{name: name, text: text}, nil
}

// Name returns the printed name, or "" for a benefit printed without one.
func (b FeatBenefit) Name() string { return b.name }

// Text returns the verbatim text.
func (b FeatBenefit) Text() string { return b.text }

func (b FeatBenefit) valid() bool {
	_, err := NewFeatBenefit(b.name, b.text)
	return err == nil
}

// FeatFields holds the members of a [Feat] beyond its identity, for
// [NewFeat]. Construct it with a keyed literal.
type FeatFields struct {
	// Category is the feat category (required).
	Category FeatCategory
	// Prerequisites lists the prerequisites; nil for none. A non-nil empty
	// list is rejected.
	Prerequisites []Prerequisite
	// Benefits is a non-empty list of benefits in printed order.
	Benefits []FeatBenefit
	// Repeatable says whether the feat may be taken more than once.
	Repeatable bool
	// RepeatableText is the printed Repeatable sentence, verbatim; required
	// when Repeatable is true and empty otherwise.
	RepeatableText string
}

// Feat is a feat (SRD 5.2.1, "Parts of a Feat", pp. 87-88). It is an [Entry]
// of kind [KindFeat] and implements [HasFeatCategory], so a background's feat
// reference can be checked for the Origin category. It has no whole
// description: the description is its benefits, prerequisites and repeatable
// text. Immutable, with its collections copied on construction and on read;
// the zero Feat is invalid. It holds slices, so it is not comparable with ==;
// compare with [Feat.Equal] or [Set.Equal]. Safe for concurrent use.
type Feat struct {
	id             ID
	name           string
	source         SourceCitation
	category       FeatCategory
	prerequisites  []Prerequisite
	benefits       []FeatBenefit
	repeatable     bool
	repeatableText string
}

var (
	_ Entry           = Feat{}
	_ HasFeatCategory = Feat{}
)

// NewFeat returns the feat id called name, with the members of f. A zero
// source means the entry cites none.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError] for the first rule broken,
// in the order of the members: BlockInvalidName; BlockInvalidMember for a
// category that is not a member; for the prerequisites BlockEmpty (a non-nil
// empty list) and BlockInvalidMember (a nil one or one that is not valid); for
// the benefits BlockEmpty and BlockInvalidMember; BlockMissing for a repeatable
// feat with no repeatable text and BlockConstraint for repeatable text on a
// feat that is not repeatable; BlockBlankText or BlockInvalidCharacter for a
// repeatable text that breaks the text rule. It never panics.
func NewFeat(id FeatID, name string, source SourceCitation, f FeatFields) (Feat, error) {
	const block = "Feat"
	pid, err := ParseID(string(id))
	if err != nil {
		return Feat{}, err
	}
	if err := checkName(block, name); err != nil {
		return Feat{}, err
	}
	if !f.Category.Valid() {
		return Feat{}, blockError(block, BlockInvalidMember)
	}
	if f.Prerequisites != nil && len(f.Prerequisites) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return Feat{}, blockError(block, BlockEmpty)
	}
	if slices.ContainsFunc(f.Prerequisites, func(p Prerequisite) bool { return !validPrerequisite(p) }) {
		return Feat{}, blockError(block, BlockInvalidMember)
	}
	if len(f.Benefits) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return Feat{}, blockError(block, BlockEmpty)
	}
	if slices.ContainsFunc(f.Benefits, func(b FeatBenefit) bool { return !b.valid() }) {
		return Feat{}, blockError(block, BlockInvalidMember)
	}
	switch {
	case f.Repeatable && f.RepeatableText == "": // equivalent mutant: a string is never below "", so == and <= agree
		return Feat{}, blockError(block, BlockMissing)
	case !f.Repeatable && f.RepeatableText != "": // equivalent mutant: a non-empty string is above "", so != and > agree
		return Feat{}, blockError(block, BlockConstraint)
	case f.RepeatableText != "": // equivalent mutant: a non-empty string is above "", so != and > agree
		if err := checkText(block, f.RepeatableText); err != nil {
			return Feat{}, err
		}
	}
	return Feat{
		id: pid, name: name, source: source, category: f.Category, prerequisites: slices.Clone(f.Prerequisites),
		benefits: slices.Clone(f.Benefits), repeatable: f.Repeatable, repeatableText: f.RepeatableText,
	}, nil
}

// MustFeat is like [NewFeat] but panics with its error when the arguments are
// invalid.
func MustFeat(id FeatID, name string, source SourceCitation, f FeatFields) Feat {
	v, err := NewFeat(id, name, source, f)
	if err != nil {
		panic(err)
	}
	return v
}

// validPrerequisite reports whether p is a prerequisite that its constructor
// would accept.
func validPrerequisite(p Prerequisite) bool {
	var err error
	switch v := p.(type) {
	case LevelPrerequisite:
		_, err = NewLevelPrerequisite(v.minimum)
	case AbilityScorePrerequisite:
		_, err = NewAbilityScorePrerequisite(v.abilities, v.minimum)
	case FeaturePrerequisite:
		_, err = NewFeaturePrerequisite(v.name)
	default:
		return false
	}
	return err == nil
}

// Key returns the key of the feat: kind [KindFeat] and its id.
func (f Feat) Key() Key { return NewKey(KindFeat, f.id) }

// ID returns the feat id.
func (f Feat) ID() FeatID { return FeatID(f.id.String()) }

// Name returns the feat name.
func (f Feat) Name() string { return f.name }

// Source returns the citation of the entry, or the zero [SourceCitation] when
// it cites none.
func (f Feat) Source() SourceCitation { return f.source }

// FeatCategory returns the category of the feat. It implements
// [HasFeatCategory].
func (f Feat) FeatCategory() FeatCategory { return f.category }

// Prerequisites returns the prerequisites as a new slice, or nil when the
// feat has none.
func (f Feat) Prerequisites() []Prerequisite { return slices.Clone(f.prerequisites) }

// Benefits returns the benefits in printed order, as a new slice.
func (f Feat) Benefits() []FeatBenefit { return slices.Clone(f.benefits) }

// Repeatable reports whether the feat may be taken more than once.
func (f Feat) Repeatable() bool { return f.repeatable }

// RepeatableText returns the printed Repeatable sentence, verbatim, or "" when
// the feat is not repeatable.
func (f Feat) RepeatableText() string { return f.repeatableText }

// MustFeatBenefit is like [NewFeatBenefit] but panics with its error when the
// arguments are invalid.
func MustFeatBenefit(name, text string) FeatBenefit {
	b, err := NewFeatBenefit(name, text)
	if err != nil {
		panic(err)
	}
	return b
}
