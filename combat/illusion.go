package combat

import (
	"slices"
	"strings"
)

// IllusionSenseID names a sense or mental faculty an illusion deceives (SRD 5.2.1,
// "Illusions", p. 183: "which senses or mental faculties it deceives"). It is an
// open vocabulary the host or the effect description supplies; IllusionSenseVisual
// is the one known value. Build it with NewIllusionSenseID; the zero value is
// invalid. Comparable and safe for concurrent use.
type IllusionSenseID string

// IllusionSenseVisual is the sight sense: an illusion that deceives it is a Visual
// Illusion (SRD 5.2.1, "Truesight", p. 190).
const IllusionSenseVisual IllusionSenseID = "visual"

// NewIllusionSenseID returns the id for s. It returns an error matching
// srderr.ErrInvalidInput for an empty or all-whitespace s. It never panics.
func NewIllusionSenseID(s string) (IllusionSenseID, error) {
	if strings.TrimSpace(s) == "" {
		return "", invalidInput("illusion sense id must not be blank")
	}
	return IllusionSenseID(s), nil
}

// IllusionBodyFacts are the physical facts of an illusion that manifests in space:
// it is insubstantial and weighs 0 pounds (SRD 5.2.1, "Illusions", p. 183). They
// are constants of the rule, so the zero value is the only value.
type IllusionBodyFacts struct{}

// Insubstantial reports true.
func (IllusionBodyFacts) Insubstantial() bool { return true }

// WeightPounds returns 0.
func (IllusionBodyFacts) WeightPounds() int { return 0 }

// EnvironmentAppearance is how an illusion in space looks under the environment.
// The zero value is AsIfReal, the default.
type EnvironmentAppearance int

// The environment appearances.
const (
	// AsIfReal: shadows, reflections, wind and echoes affect the illusion as if it were real (the default).
	AsIfReal EnvironmentAppearance = iota
	// SpecifiedByEffect: the creating effect says otherwise.
	SpecifiedByEffect
)

// String returns "as_if_real", "specified_by_effect" or "unknown".
func (a EnvironmentAppearance) String() string {
	switch a {
	case AsIfReal:
		return "as_if_real"
	case SpecifiedByEffect:
		return "specified_by_effect"
	}
	return "unknown"
}

// IllusionManifestation is the closed set of whether an illusion manifests in
// space. Variants: NotInSpace, ManifestsInSpace. Only ManifestsInSpace declares
// body facts and an appearance, so no nil value stands in.
type IllusionManifestation interface{ isIllusionManifestation() }

// NotInSpace: the illusion does not manifest in space, so it has no body facts and
// no environment appearance.
type NotInSpace struct{}

// ManifestsInSpace: the illusion manifests in space (SRD 5.2.1, "Illusions", p.
// 183). Build it with NewManifestsInSpace; its zero value is a valid illusion that
// appears as if real.
type ManifestsInSpace struct{ appearance EnvironmentAppearance }

// NewManifestsInSpace returns the manifestation with the appearance. It returns an
// error matching srderr.ErrInvalidInput for an undefined appearance. It never panics.
func NewManifestsInSpace(appearance EnvironmentAppearance) (ManifestsInSpace, error) {
	if appearance != AsIfReal && appearance != SpecifiedByEffect {
		return ManifestsInSpace{}, invalidInput("undefined environment appearance")
	}
	return ManifestsInSpace{appearance}, nil
}

// BodyFacts returns the insubstantial, weightless facts of the illusion.
func (ManifestsInSpace) BodyFacts() IllusionBodyFacts { return IllusionBodyFacts{} }

// Appearance returns how the environment affects the illusion.
func (m ManifestsInSpace) Appearance() EnvironmentAppearance { return m.appearance }

func (NotInSpace) isIllusionManifestation()       {}
func (ManifestsInSpace) isIllusionManifestation() {}

// IllusionTraits are the facts of an illusion that the engine uses: the senses it
// deceives and whether it manifests in space (SRD 5.2.1, "Illusions", p. 183). Build
// it with NewIllusionTraits; the zero value is invalid. It holds a slice, so it is
// not comparable with ==. Safe for concurrent use.
type IllusionTraits struct {
	senses        []IllusionSenseID
	manifestation IllusionManifestation
}

// NewIllusionTraits returns the traits. senses is copied, repeats are kept once and
// the first-seen order is kept. It returns an error matching srderr.ErrInvalidInput
// for an empty set (package design: an illusion that deceives nothing is not an
// illusion), a blank sense id, or a nil manifestation, and a LimitExceededError for more than 256 senses. It never panics.
func NewIllusionTraits(senses []IllusionSenseID, manifestation IllusionManifestation) (IllusionTraits, error) {
	if err := overCap("illusion senses", len(senses), maxInputList); err != nil {
		return IllusionTraits{}, err
	}
	if len(senses) == 0 || manifestation == nil {
		return IllusionTraits{}, invalidInput("an illusion deceives at least one sense and has a manifestation")
	}
	var set []IllusionSenseID
	for _, s := range senses {
		if strings.TrimSpace(string(s)) == "" {
			return IllusionTraits{}, invalidInput("illusion sense id must not be blank")
		}
		if !slices.Contains(set, s) {
			set = append(set, s)
		}
	}
	return IllusionTraits{set, manifestation}, nil
}

// Senses returns a copy of the senses the illusion deceives.
func (t IllusionTraits) Senses() []IllusionSenseID { return slices.Clone(t.senses) }

// Manifestation returns NotInSpace or ManifestsInSpace.
func (t IllusionTraits) Manifestation() IllusionManifestation { return t.manifestation }

// DeceivesSight reports whether the visual sense is among the senses: the Visual
// Illusion predicate of the senses rules (SRD 5.2.1, "Truesight", p. 190). Another
// sense, such as a made-up hearing, never makes it true.
func (t IllusionTraits) DeceivesSight() bool { return slices.Contains(t.senses, IllusionSenseVisual) }
