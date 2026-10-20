package content

import (
	"math"
	"slices"
	"strconv"
)

// The limits of a [Species] header, shared by the constructor and the decoder.
const (
	// SpeciesMaxSizes is the most sizes a species lists: one, or two when the
	// player chooses (SRD 5.2.1, "Creature Type and Size", p. 83).
	SpeciesMaxSizes = 2
	// SpeciesMaxSpeed is the highest speed in feet a species may have. The
	// SRD sets none; the bound keeps the value the same on every platform
	// integer width.
	SpeciesMaxSpeed = math.MaxInt32
)

// sizeAllowed reports whether z is a size a species may list: Small or
// Medium (SRD 5.2.1, "Parts of a Species: Size", p. 83).
func sizeAllowed(z CreatureSize) bool { return z == SizeSmall || z == SizeMedium }

// SpeciesFields holds the members of a [Species] beyond its identity, for
// [NewSpecies]. Construct it with a keyed literal.
type SpeciesFields struct {
	// CreatureType is the creature type (required).
	CreatureType CreatureType
	// Sizes lists one or two distinct sizes, each Small or Medium.
	Sizes []CreatureSize
	// SizeText is the printed Size sentence, verbatim; empty when absent.
	SizeText string
	// Speed is the speed in feet, 0 to [SpeciesMaxSpeed].
	Speed int
	// Traits is a non-empty list of special traits with distinct names.
	Traits []NamedText
	// Choices lists the traits that offer a closed list of options; possibly
	// none.
	Choices []SpeciesChoice
}

// Species is a character species (SRD 5.2.1, "Character Species", pp. 83-86).
// It is an [Entry] of kind [KindSpecies] and implements [References]: the
// spells its choice options name. Immutable, with its collections copied on
// construction and on read; the zero Species is invalid. It holds slices, so
// it is not comparable with ==; compare with [Species.Equal] or [Set.Equal]. Safe for
// concurrent use.
type Species struct {
	id           ID
	name         string
	source       SourceCitation
	creatureType CreatureType
	sizes        []CreatureSize
	sizeText     string
	speed        int
	traits       []NamedText
	choices      []SpeciesChoice
}

var (
	_ Entry      = Species{}
	_ References = Species{}
)

// NewSpecies returns the species id called name, with the members of f. A
// zero source means the entry cites none.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError] for the first rule broken,
// in the order of the members: BlockInvalidName; BlockInvalidMember for a
// creature type that is not a member; for the sizes BlockOutOfRange (not one
// or two), BlockInvalidMember (not Small or Medium) and BlockDuplicate; for
// the size text BlockBlankText or BlockInvalidCharacter; BlockOutOfRange for a
// speed outside 0 to [SpeciesMaxSpeed]; for the traits BlockEmpty,
// BlockInvalidMember (a trait that is not valid) and BlockDuplicate (a repeated
// name); for the choices BlockInvalidMember and BlockConstraint (a trait name
// no trait has). It never panics.
func NewSpecies(id SpeciesID, name string, source SourceCitation, f SpeciesFields) (Species, error) {
	const block = "Species"
	pid, err := ParseID(string(id))
	if err != nil {
		return Species{}, err
	}
	if err := checkName(block, name); err != nil {
		return Species{}, err
	}
	if !f.CreatureType.Valid() {
		return Species{}, blockError(block, BlockInvalidMember)
	}
	if err := checkSet(block, f.Sizes, 1, SpeciesMaxSizes, sizeAllowed); err != nil {
		return Species{}, err
	}
	if f.SizeText != "" { // equivalent mutant: a non-empty string is above "", so != and > agree
		if err := checkText(block, f.SizeText); err != nil {
			return Species{}, err
		}
	}
	if f.Speed < 0 || f.Speed > SpeciesMaxSpeed {
		return Species{}, blockError(block, BlockOutOfRange)
	}
	if err := checkNamedTexts(block, f.Traits); err != nil {
		return Species{}, err
	}
	for _, c := range f.Choices {
		if !c.valid() {
			return Species{}, blockError(block, BlockInvalidMember)
		}
		if !slices.ContainsFunc(f.Traits, func(t NamedText) bool { return t.name == c.traitName }) {
			return Species{}, blockError(block, BlockConstraint)
		}
	}
	return Species{
		id: pid, name: name, source: source, creatureType: f.CreatureType, sizes: slices.Clone(f.Sizes),
		sizeText: f.SizeText, speed: f.Speed, traits: slices.Clone(f.Traits), choices: cloneOrNil(f.Choices),
	}, nil
}

// MustSpecies is like [NewSpecies] but panics with its error when the
// arguments are invalid.
func MustSpecies(id SpeciesID, name string, source SourceCitation, f SpeciesFields) Species {
	s, err := NewSpecies(id, name, source, f)
	if err != nil {
		panic(err)
	}
	return s
}

// checkSet checks a list that must be a set of valid members: between lo and
// hi long (BlockOutOfRange), each accepted by valid (BlockInvalidMember) and
// none repeated (BlockDuplicate), checked in that order per element.
func checkSet[T comparable](block string, items []T, lo, hi int, valid func(T) bool) error {
	if len(items) < lo || len(items) > hi {
		return blockError(block, BlockOutOfRange)
	}
	seen := make(map[T]struct{}, len(items))
	for _, v := range items {
		if !valid(v) {
			return blockError(block, BlockInvalidMember)
		}
		if _, dup := seen[v]; dup {
			return blockError(block, BlockDuplicate)
		}
		seen[v] = struct{}{}
	}
	return nil
}

// checkNamedTexts checks a non-empty list of valid NamedTexts with distinct
// names: BlockEmpty, BlockInvalidMember, BlockDuplicate.
func checkNamedTexts(block string, items []NamedText) error {
	if len(items) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return blockError(block, BlockEmpty)
	}
	names := make(map[string]struct{}, len(items))
	for _, t := range items {
		if !validName(t.name) || !validText(t.text) {
			return blockError(block, BlockInvalidMember)
		}
		if _, repeated := names[t.name]; repeated {
			return blockError(block, BlockDuplicate)
		}
		names[t.name] = struct{}{}
	}
	return nil
}

// Key returns the key of the species: kind [KindSpecies] and its id.
func (s Species) Key() Key { return NewKey(KindSpecies, s.id) }

// ID returns the species id.
func (s Species) ID() SpeciesID { return SpeciesID(s.id.String()) }

// Name returns the species name.
func (s Species) Name() string { return s.name }

// Source returns the citation of the entry, or the zero [SourceCitation] when
// it cites none.
func (s Species) Source() SourceCitation { return s.source }

// CreatureType returns the creature type.
func (s Species) CreatureType() CreatureType { return s.creatureType }

// Sizes returns the one or two sizes the species lists, as a new slice.
func (s Species) Sizes() []CreatureSize { return slices.Clone(s.sizes) }

// SizeText returns the printed Size sentence, verbatim, or "" when the entry
// has none.
func (s Species) SizeText() string { return s.sizeText }

// Speed returns the speed in feet.
func (s Species) Speed() int { return s.speed }

// Traits returns the special traits in printed order, as a new slice.
func (s Species) Traits() []NamedText { return slices.Clone(s.traits) }

// Choices returns the traits that offer options, possibly none, as a new
// slice.
func (s Species) Choices() []SpeciesChoice { return slices.Clone(s.choices) }

// References returns the spells the choice options name, one site per spell
// id, in the order of the choices, options and entries (FR-023): the fields
// "choices/<j>/options/<k>/levelBenefits/<m>/spellId" and
// "choices/<j>/options/<k>/spellIds/<n>". The target of each is the spell of
// kind [KindSpell] with no constraint.
func (s Species) References() []Reference {
	var refs []Reference
	for j, c := range s.choices {
		for k, o := range c.options {
			base := "choices/" + strconv.Itoa(j) + "/options/" + strconv.Itoa(k)
			switch o := o.(type) {
			case LineageOption:
				for m, b := range o.benefits {
					refs = append(refs, MustReference(base+"/levelBenefits/"+strconv.Itoa(m)+"/spellId", b.spellID.Key(), nil))
				}
			case TextOption:
				for n, id := range o.spellIDs {
					refs = append(refs, MustReference(base+"/spellIds/"+strconv.Itoa(n), id.Key(), nil))
				}
			}
		}
	}
	return refs
}
