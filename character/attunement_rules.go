package character

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
)

// PrerequisiteJudgement is the host's reading of a described Attunement
// prerequisite the engine cannot evaluate. The zero value is PrerequisiteNotSupplied.
type PrerequisiteJudgement int

// The host's judgements.
const (
	PrerequisiteNotSupplied PrerequisiteJudgement = iota
	PrerequisiteMet
	PrerequisiteNotMet
)

// OwnSpellcasting is whether a creature can cast at least one spell using its
// own traits or features, not by using a magic item or the like (SRD 5.2.1,
// "Magic Item Rules", p. 205). The zero value is invalid.
type OwnSpellcasting int

// Whether the creature has spellcasting of its own.
const (
	CanCast OwnSpellcasting = iota + 1
	CannotCast
)

// AttuneCandidate is the creature that wants to attune: the classes in which it
// has a level of 1 or more, and whether it has spellcasting of its own.
// Immutable. The zero value has no classes and invalid spellcasting.
type AttuneCandidate struct {
	classes      []content.ClassID
	spellcasting OwnSpellcasting
}

// NewAttuneCandidate returns the candidate, copying classes. It returns an
// error matching srderr.ErrInvalidInput for an invalid class id or spellcasting.
func NewAttuneCandidate(classes []content.ClassID, spellcasting OwnSpellcasting) (AttuneCandidate, error) {
	if spellcasting != CanCast && spellcasting != CannotCast || slices.ContainsFunc(classes, func(c content.ClassID) bool { return !c.Valid() }) {
		return AttuneCandidate{}, invalidInput("attune candidate: invalid class or spellcasting")
	}
	return AttuneCandidate{classes: slices.Clone(classes), spellcasting: spellcasting}, nil
}

// Classes returns a new slice of the classes the creature has levels in.
func (c AttuneCandidate) Classes() []content.ClassID { return slices.Clone(c.classes) }

// Spellcasting returns whether the creature can cast spells of its own.
func (c AttuneCandidate) Spellcasting() OwnSpellcasting { return c.spellcasting }

// AttunementPrerequisiteReader maps an item's Attunement requirement to the
// prerequisite the rules evaluate. The zero value is ready to use.
type AttunementPrerequisiteReader struct{}

func srdClassNames() []string {
	return []string{"barbarian", "bard", "cleric", "druid", "fighter", "monk", "paladin", "ranger", "rogue", "sorcerer", "warlock", "wizard"}
}

// Read maps attunement: AttunementRequired is NoPrerequisite; "a Spellcaster" is
// SpellcasterPrerequisite; "a <Class>" or a list of SRD class names joined by
// commas and a final "or" is ClassPrerequisite; any other text is a
// DescribedPrerequisite, verbatim. It returns an error matching
// srderr.ErrInvalidInput for content.NoAttunement or a nil attunement.
func (AttunementPrerequisiteReader) Read(attunement content.Attunement) (content.AttunementPrerequisite, error) {
	switch a := attunement.(type) {
	case content.AttunementRequired:
		return content.NoPrerequisite{}, nil
	case content.AttunementRequiredBy:
		return readPrerequisite(a.Prerequisite())
	default:
		return nil, invalidInput("read prerequisite: no attunement or nil attunement")
	}
}

func readPrerequisite(text string) (content.AttunementPrerequisite, error) {
	if text == "a Spellcaster" {
		return content.SpellcasterPrerequisite{}, nil
	}
	list := strings.TrimPrefix(strings.TrimPrefix(text, "a "), "an ")
	list = strings.ReplaceAll(strings.ReplaceAll(list, ", or ", ","), " or ", ",")
	var classes []content.ClassID
	for _, name := range strings.Split(list, ",") {
		name = strings.ToLower(strings.TrimSpace(name))
		if !slices.Contains(srdClassNames(), name) {
			return content.NewDescribedPrerequisite(text)
		}
		classes = append(classes, content.MustClassID(name))
	}
	return content.NewClassPrerequisite(classes)
}

// AttunementEligibility evaluates Attunement prerequisites. The zero value is
// ready to use.
type AttunementEligibility struct{}

// IsMet reports whether candidate satisfies prerequisite: NoPrerequisite always;
// ClassPrerequisite when the creature has a level in a named class;
// SpellcasterPrerequisite when it CanCast; DescribedPrerequisite when judgement
// is PrerequisiteMet (PrerequisiteNotSupplied is not met, see JudgementMissing).
// It returns an error matching srderr.ErrInvalidInput for an empty class
// prerequisite or an unknown prerequisite type.
func (AttunementEligibility) IsMet(prerequisite content.AttunementPrerequisite, candidate AttuneCandidate, judgement PrerequisiteJudgement) (bool, error) {
	switch p := prerequisite.(type) {
	case content.NoPrerequisite:
		return true, nil
	case content.ClassPrerequisite:
		named := p.Classes()
		if len(named) == 0 {
			return false, invalidInput("class prerequisite names no class")
		}
		return slices.ContainsFunc(candidate.classes, func(c content.ClassID) bool { return slices.Contains(named, c) }), nil
	case content.SpellcasterPrerequisite:
		return candidate.spellcasting == CanCast, nil
	case content.DescribedPrerequisite:
		return judgement == PrerequisiteMet, nil
	default:
		return false, invalidInput("nil or foreign attunement prerequisite")
	}
}

// JudgementMissing reports whether prerequisite is a described one the host has
// not judged.
func (AttunementEligibility) JudgementMissing(prerequisite content.AttunementPrerequisite, judgement PrerequisiteJudgement) bool {
	_, described := prerequisite.(content.DescribedPrerequisite)
	return described && judgement == PrerequisiteNotSupplied
}

// AttunementEndingVerdict is the closed set of answers to "does Attunement end".
// Variants: AttunementEnds, AttunementContinues.
type AttunementEndingVerdict interface{ isAttunementEndingVerdict() }

// AttunementEnds: Attunement ends for Reason.
type AttunementEnds struct{ Reason AttunementEndReason }

// AttunementContinues: Attunement does not end.
type AttunementContinues struct{}

func (AttunementEnds) isAttunementEndingVerdict()      {}
func (AttunementContinues) isAttunementEndingVerdict() {}

// AttunementEnding decides the ending conditions of Attunement. The zero value
// is ready to use.
type AttunementEnding struct{}

// CheckPrerequisite ends Attunement when the creature no longer meets the
// prerequisite. A described prerequisite the host has not judged continues: the
// engine never ends Attunement on a prerequisite it cannot evaluate. It returns
// the errors of AttunementEligibility.IsMet.
func (AttunementEnding) CheckPrerequisite(prerequisite content.AttunementPrerequisite, candidate AttuneCandidate, judgement PrerequisiteJudgement) (AttunementEndingVerdict, error) {
	e := AttunementEligibility{}
	if e.JudgementMissing(prerequisite, judgement) {
		return AttunementContinues{}, nil
	}
	met, err := e.IsMet(prerequisite, candidate, judgement)
	if err != nil {
		return nil, err
	}
	if met {
		return AttunementContinues{}, nil
	}
	return AttunementEnds{Reason: AttunementEndedPrerequisiteNoLongerMet}, nil
}

// CheckDistance ends Attunement when the item has been more than 100 feet away
// for at least 24 hours (SRD 5.2.1, "Ending Attunement", p. 102). It returns an
// error matching srderr.ErrInvalidInput for a negative distance or hours.
func (AttunementEnding) CheckDistance(distanceFeet, hoursAway int) (AttunementEndingVerdict, error) {
	if distanceFeet < 0 || hoursAway < 0 {
		return nil, invalidInput("negative distance or hours away")
	}
	f := content.AttunementFacts{}
	if distanceFeet > f.EndingDistanceFeet() && hoursAway >= f.EndingHoursAway() {
		return AttunementEnds{Reason: AttunementEndedItemTooFarAway}, nil
	}
	return AttunementContinues{}, nil
}

// AttunementStanding is a creature's standing towards an item. The zero value
// is invalid.
type AttunementStanding int

// The standings.
const (
	StandingAttuned AttunementStanding = iota + 1
	StandingNotAttuned
)

// MagicalPropertiesAccessLevel is what an item's properties give a creature.
// The zero value is invalid.
type MagicalPropertiesAccessLevel int

// The access levels.
const (
	AccessFullMagical MagicalPropertiesAccessLevel = iota + 1
	AccessNonmagicalBenefitsOnly
	AccessNoMagic
)

// MagicalPropertiesAccess decides what a creature gets from an item. The zero
// value is ready to use.
type MagicalPropertiesAccess struct{}

// Of returns AccessNoMagic for a consumed item; AccessFullMagical for an item
// needing no Attunement, one whose standing is StandingAttuned, or a property the
// description says works without Attunement; and AccessNonmagicalBenefitsOnly
// otherwise (SRD 5.2.1, "Attunement", p. 102).
func (MagicalPropertiesAccess) Of(requirement content.Attunement, standing AttunementStanding, availableWithoutAttunement bool, use MagicItemUseState) MagicalPropertiesAccessLevel {
	if use == ItemConsumed {
		return AccessNoMagic
	}
	if _, none := requirement.(content.NoAttunement); none || standing == StandingAttuned || availableWithoutAttunement {
		return AccessFullMagical
	}
	return AccessNonmagicalBenefitsOnly
}
