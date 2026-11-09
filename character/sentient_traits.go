package character

import (
	"strings"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// SentientAbilityRoll rolls a sentient item's ability scores. The zero value is
// ready to use.
type SentientAbilityRoll struct{}

// FromDice returns the score of one 4d6 roll with the lowest die dropped, from 3
// to 18 (SRD 5.2.1, "Abilities", p. 207). It returns an error matching
// srderr.ErrInvalidInput for a result that is not four dice.
func (SentientAbilityRoll) FromDice(r dice.Result) (ability.Score, error) {
	if len(r.Rolls()) != 4 {
		return ability.Score{}, invalidInput("sentient ability roll: not four dice")
	}
	sum, err := r.SumOfHighest(3)
	if err != nil {
		return ability.Score{}, err
	}
	return ability.NewScore(sum)
}

// DescribedTarget is the verbatim, non-blank target of a Protector or Templar
// purpose; the SRD leaves it open. The zero value is invalid.
type DescribedTarget string

// NewDescribedTarget returns text as a target. It returns an error matching
// srderr.ErrInvalidInput for blank text.
func NewDescribedTarget(text string) (DescribedTarget, error) {
	if strings.TrimSpace(text) == "" {
		return "", invalidInput("blank described target")
	}
	return DescribedTarget(text), nil
}

// SpecialPurpose is the closed set of special purposes of a sentient item (SRD
// 5.2.1, "Special Purpose", p. 208). Variants: AlignedPurpose, BanePurpose,
// CreatorSeeker, DestinySeeker, Destroyer, GlorySeeker, LoreSeeker,
// ProtectorPurpose, SoulmateSeeker, TemplarPurpose.
type SpecialPurpose interface {
	Kind() content.SpecialPurposeKind
}

// AlignedPurposeReason says why an Aligned purpose was refused. The zero value is not a reason.
type AlignedPurposeReason int

// The reasons.
const (
	AlignedItemIsNeutral AlignedPurposeReason = iota + 1
	AlignedNotDiametricallyOpposed
)

// AlignedPurposeRejected is the error NewAlignedPurpose returns. It matches
// srderr.ErrInvalidInput.
//
//nolint:errname // the name is the spec's (FR-119); it is an error with typed data
type AlignedPurposeRejected struct{ Reason AlignedPurposeReason }

// Error renders the error. The text is not a contract.
func (e *AlignedPurposeRejected) Error() string { return "character: aligned purpose rejected" }

// Unwrap returns srderr.ErrInvalidInput.
func (e *AlignedPurposeRejected) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is an *AlignedPurposeRejected with the same Reason.
func (e *AlignedPurposeRejected) Is(target error) bool {
	t, ok := target.(*AlignedPurposeRejected)
	return ok && t != nil && e != nil && *t == *e
}

// AlignedPurpose is the purpose of destroying an alignment opposed to the item's.
type AlignedPurpose struct{ opposed content.Alignment }

// axes returns the law and good axes of a, each -1, 0 or 1, and false for unaligned or unknown.
func axes(a content.Alignment) (law, good int, ok bool) {
	if a < content.AlignmentLawfulGood || a > content.AlignmentChaoticEvil {
		return 0, 0, false
	}
	i := int(a - content.AlignmentLawfulGood)
	return i/3 - 1, 1 - i%3, true
}

// NewAlignedPurpose returns the purpose. The item MUST NOT be Neutral ("such an
// item is never Neutral") and opposed MUST be diametrically opposed to it, the
// opposite on both axes where both exist (Lawful Good against Chaotic Evil; an
// interpretation of the SRD's wording). It returns an *AlignedPurposeRejected
// (matching srderr.ErrInvalidInput) with AlignedItemIsNeutral or
// AlignedNotDiametricallyOpposed. It never panics.
func NewAlignedPurpose(itemAlignment, opposed content.Alignment) (AlignedPurpose, error) {
	law, good, ok := axes(itemAlignment)
	if ok && law == 0 && good == 0 {
		return AlignedPurpose{}, &AlignedPurposeRejected{Reason: AlignedItemIsNeutral}
	}
	oLaw, oGood, oOK := axes(opposed)
	if !ok || !oOK || oLaw != -law || oGood != -good {
		return AlignedPurpose{}, &AlignedPurposeRejected{Reason: AlignedNotDiametricallyOpposed}
	}
	return AlignedPurpose{opposed: opposed}, nil
}

// MustAlignedPurpose is NewAlignedPurpose for literals; it panics with the error.
func MustAlignedPurpose(itemAlignment, opposed content.Alignment) AlignedPurpose {
	p, err := NewAlignedPurpose(itemAlignment, opposed)
	if err != nil {
		panic(err)
	}
	return p
}

// Opposed returns the alignment the item opposes.
func (p AlignedPurpose) Opposed() content.Alignment { return p.opposed }

// Kind returns content.SpecialPurposeAligned.
func (AlignedPurpose) Kind() content.SpecialPurposeKind { return content.SpecialPurposeAligned }

// BanePurpose is the purpose of destroying a creature type.
type BanePurpose struct{ creature content.CreatureType }

// NewBanePurpose returns the purpose. It returns an error matching
// srderr.ErrInvalidInput for an invalid creature type.
func NewBanePurpose(t content.CreatureType) (BanePurpose, error) {
	if !t.Valid() {
		return BanePurpose{}, invalidInput("bane purpose: invalid creature type")
	}
	return BanePurpose{creature: t}, nil
}

// CreatureType returns the creature type.
func (p BanePurpose) CreatureType() content.CreatureType { return p.creature }

// Kind returns content.SpecialPurposeBane.
func (BanePurpose) Kind() content.SpecialPurposeKind { return content.SpecialPurposeBane }

// CreatorSeeker is the purpose of finding the item's creator.
type CreatorSeeker struct{}

// DestinySeeker is the purpose of fulfilling a destiny.
type DestinySeeker struct{}

// Destroyer is the purpose of destruction.
type Destroyer struct{}

// GlorySeeker is the purpose of seeking glory.
type GlorySeeker struct{}

// LoreSeeker is the purpose of seeking lore.
type LoreSeeker struct{}

// SoulmateSeeker is the purpose of seeking a soulmate.
type SoulmateSeeker struct{}

// ProtectorPurpose is the purpose of protecting a described target.
type ProtectorPurpose struct{ target DescribedTarget }

// NewProtectorPurpose returns the purpose. It returns an error matching
// srderr.ErrInvalidInput for a blank target.
func NewProtectorPurpose(t DescribedTarget) (ProtectorPurpose, error) {
	if strings.TrimSpace(string(t)) == "" {
		return ProtectorPurpose{}, invalidInput("protector purpose: blank target")
	}
	return ProtectorPurpose{target: t}, nil
}

// Target returns the protected target.
func (p ProtectorPurpose) Target() DescribedTarget { return p.target }

// TemplarPurpose is the purpose of serving a described target.
type TemplarPurpose struct{ target DescribedTarget }

// NewTemplarPurpose returns the purpose. It returns an error matching
// srderr.ErrInvalidInput for a blank target.
func NewTemplarPurpose(t DescribedTarget) (TemplarPurpose, error) {
	if strings.TrimSpace(string(t)) == "" {
		return TemplarPurpose{}, invalidInput("templar purpose: blank target")
	}
	return TemplarPurpose{target: t}, nil
}

// Target returns the served target.
func (p TemplarPurpose) Target() DescribedTarget { return p.target }

// Kind returns content.SpecialPurposeCreatorSeeker.
func (CreatorSeeker) Kind() content.SpecialPurposeKind { return content.SpecialPurposeCreatorSeeker }

// Kind returns content.SpecialPurposeDestinySeeker.
func (DestinySeeker) Kind() content.SpecialPurposeKind { return content.SpecialPurposeDestinySeeker }

// Kind returns content.SpecialPurposeDestroyer.
func (Destroyer) Kind() content.SpecialPurposeKind { return content.SpecialPurposeDestroyer }

// Kind returns content.SpecialPurposeGlorySeeker.
func (GlorySeeker) Kind() content.SpecialPurposeKind { return content.SpecialPurposeGlorySeeker }

// Kind returns content.SpecialPurposeLoreSeeker.
func (LoreSeeker) Kind() content.SpecialPurposeKind { return content.SpecialPurposeLoreSeeker }

// Kind returns content.SpecialPurposeSoulmateSeeker.
func (SoulmateSeeker) Kind() content.SpecialPurposeKind { return content.SpecialPurposeSoulmateSeeker }

// Kind returns content.SpecialPurposeProtector.
func (ProtectorPurpose) Kind() content.SpecialPurposeKind { return content.SpecialPurposeProtector }

// Kind returns content.SpecialPurposeTemplar.
func (TemplarPurpose) Kind() content.SpecialPurposeKind { return content.SpecialPurposeTemplar }

// SpecialPurposeSelection is no special purpose or one. The zero value is no purpose.
type SpecialPurposeSelection struct{ purpose SpecialPurpose }

// NoSpecialPurpose is the selection of no purpose.
func NoSpecialPurpose() SpecialPurposeSelection { return SpecialPurposeSelection{} }

// SelectPurpose is the selection of purpose.
func SelectPurpose(p SpecialPurpose) SpecialPurposeSelection {
	return SpecialPurposeSelection{purpose: p}
}

// Purpose returns the purpose and true, or nil and false for no purpose.
func (s SpecialPurposeSelection) Purpose() (SpecialPurpose, bool) { return s.purpose, s.purpose != nil }

// SentientTraitsReason says why sentient traits were refused. The zero value is not a reason.
type SentientTraitsReason int

// The reasons.
const (
	SentientSingleUseItemNotSentient SentientTraitsReason = iota + 1
	SentientUnalignedNotAllowed
	SentientAlignedPurposeInvalid
)

// String returns the snake_case name of the reason, or "unknown".
func (r SentientTraitsReason) String() string {
	names := []string{"single_use_item_not_sentient", "unaligned_not_allowed", "aligned_purpose_invalid"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// SentientTraitsRejected is the error returned for traits that cannot be
// sentient. It matches srderr.ErrInvalidInput.
//
//nolint:errname // the name is the spec's (FR-119); it is an error with typed data
type SentientTraitsRejected struct{ Reason SentientTraitsReason }

// Error renders the error. The text is not a contract.
func (e *SentientTraitsRejected) Error() string {
	return "character: sentient traits: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *SentientTraitsRejected) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *SentientTraitsRejected with the same Reason.
func (e *SentientTraitsRejected) Is(target error) bool {
	t, ok := target.(*SentientTraitsRejected)
	return ok && t != nil && e != nil && *t == *e
}

// AttachSentientTraits checks that an item of category may be sentient: every
// category but potion and scroll, which are single-use and "never sentient". It
// returns a *SentientTraitsRejected with SentientSingleUseItemNotSentient
// otherwise.
func AttachSentientTraits(category content.MagicItemCategory) error {
	if category == content.MagicItemPotion || category == content.MagicItemScroll {
		return &SentientTraitsRejected{Reason: SentientSingleUseItemNotSentient}
	}
	return nil
}

// SentientTraits are the traits of a sentient item: Intelligence, Wisdom and
// Charisma, an Alignment, communication, senses and a special purpose. Immutable
// and comparable; the zero value is invalid.
type SentientTraits struct {
	intelligence, wisdom, charisma ability.Score
	alignment                      content.Alignment
	communication                  content.SentientCommunication
	senses                         content.SentientSenses
	purpose                        SpecialPurposeSelection
}

// NewSentientTraits returns the traits of an item of category. It returns a
// *SentientTraitsRejected (matching srderr.ErrInvalidInput) with
// SentientSingleUseItemNotSentient for a potion or scroll,
// SentientUnalignedNotAllowed for an unaligned or unknown alignment, and
// SentientAlignedPurposeInvalid for an Aligned purpose that does not pass
// NewAlignedPurpose; and an error matching srderr.ErrInvalidInput for an unset
// communication or senses.
func NewSentientTraits(category content.MagicItemCategory, intelligence, wisdom, charisma ability.Score, alignment content.Alignment,
	communication content.SentientCommunication, senses content.SentientSenses, purpose SpecialPurposeSelection) (SentientTraits, error) {
	if err := AttachSentientTraits(category); err != nil {
		return SentientTraits{}, err
	}
	if _, _, ok := axes(alignment); !ok {
		return SentientTraits{}, &SentientTraitsRejected{Reason: SentientUnalignedNotAllowed}
	}
	if !communication.Valid() || senses.RangeFeet() == 0 {
		return SentientTraits{}, invalidInput("sentient traits: unset communication or senses")
	}
	if p, ok := purpose.Purpose(); ok {
		if a, aligned := p.(AlignedPurpose); aligned {
			if _, err := NewAlignedPurpose(alignment, a.opposed); err != nil {
				return SentientTraits{}, &SentientTraitsRejected{Reason: SentientAlignedPurposeInvalid}
			}
		}
	}
	return SentientTraits{intelligence: intelligence, wisdom: wisdom, charisma: charisma, alignment: alignment,
		communication: communication, senses: senses, purpose: purpose}, nil
}

// Intelligence returns the Intelligence score.
func (t SentientTraits) Intelligence() ability.Score { return t.intelligence }

// Wisdom returns the Wisdom score.
func (t SentientTraits) Wisdom() ability.Score { return t.wisdom }

// Charisma returns the Charisma score.
func (t SentientTraits) Charisma() ability.Score { return t.charisma }

// Alignment returns the alignment, never unaligned.
func (t SentientTraits) Alignment() content.Alignment { return t.alignment }

// Communication returns how the item communicates.
func (t SentientTraits) Communication() content.SentientCommunication { return t.communication }

// Senses returns the item's senses.
func (t SentientTraits) Senses() content.SentientSenses { return t.senses }

// SpecialPurpose returns the special purpose selection.
func (t SentientTraits) SpecialPurpose() SpecialPurposeSelection { return t.purpose }

// IsZero reports whether t is the invalid zero SentientTraits.
func (t SentientTraits) IsZero() bool { return t.alignment == 0 }

func (AlignedPurpose) isSpecialPurpose()   {} //nolint:unused // marker of the sealed SpecialPurpose set
func (BanePurpose) isSpecialPurpose()      {} //nolint:unused // marker of the sealed SpecialPurpose set
func (CreatorSeeker) isSpecialPurpose()    {} //nolint:unused // marker of the sealed SpecialPurpose set
func (DestinySeeker) isSpecialPurpose()    {} //nolint:unused // marker of the sealed SpecialPurpose set
func (Destroyer) isSpecialPurpose()        {} //nolint:unused // marker of the sealed SpecialPurpose set
func (GlorySeeker) isSpecialPurpose()      {} //nolint:unused // marker of the sealed SpecialPurpose set
func (LoreSeeker) isSpecialPurpose()       {} //nolint:unused // marker of the sealed SpecialPurpose set
func (ProtectorPurpose) isSpecialPurpose() {} //nolint:unused // marker of the sealed SpecialPurpose set
func (SoulmateSeeker) isSpecialPurpose()   {} //nolint:unused // marker of the sealed SpecialPurpose set
func (TemplarPurpose) isSpecialPurpose()   {} //nolint:unused // marker of the sealed SpecialPurpose set

// purposeOfKind builds the purpose of kind from detail; false when the kind
// needs a detail that is missing or invalid. An Aligned purpose is built from the
// detail alone and checked against the alignment by NewSentientTraits.
func purposeOfKind(kind content.SpecialPurposeKind, _ content.Alignment, d PurposeDetail) (SpecialPurposeSelection, bool) {
	var p SpecialPurpose
	switch kind {
	case content.SpecialPurposeAligned:
		if !d.Opposed.Valid() {
			return SpecialPurposeSelection{}, false
		}
		p = AlignedPurpose{opposed: d.Opposed}
	case content.SpecialPurposeBane:
		b, err := NewBanePurpose(d.Creature)
		if err != nil {
			return SpecialPurposeSelection{}, false
		}
		p = b
	case content.SpecialPurposeProtector:
		pr, err := NewProtectorPurpose(d.Target)
		if err != nil {
			return SpecialPurposeSelection{}, false
		}
		p = pr
	case content.SpecialPurposeTemplar:
		tp, err := NewTemplarPurpose(d.Target)
		if err != nil {
			return SpecialPurposeSelection{}, false
		}
		p = tp
	case content.SpecialPurposeCreatorSeeker:
		p = CreatorSeeker{}
	case content.SpecialPurposeDestinySeeker:
		p = DestinySeeker{}
	case content.SpecialPurposeDestroyer:
		p = Destroyer{}
	case content.SpecialPurposeGlorySeeker:
		p = GlorySeeker{}
	case content.SpecialPurposeLoreSeeker:
		p = LoreSeeker{}
	case content.SpecialPurposeSoulmateSeeker:
		p = SoulmateSeeker{}
	default:
		return SpecialPurposeSelection{}, false
	}
	return SelectPurpose(p), true
}
