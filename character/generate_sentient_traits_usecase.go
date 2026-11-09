package character

import (
	"context"
	"errors"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// PurposeDetail is the detail a special purpose may need: Opposed for Aligned,
// Creature for Bane, Target for Protector and Templar. The SRD leaves the target
// open, so the host supplies it. Construct it with a keyed literal.
type PurposeDetail struct {
	Opposed  content.Alignment
	Creature content.CreatureType
	Target   DescribedTarget
}

// GenerateSentientItemTraitsCommand is the input of GenerateSentientItemTraits.Do.
// Each trait is rolled when its Roll flag is set and otherwise taken from the
// chosen value beside it. Purpose and PurposeDetail: a rolled purpose is built
// from the detail. Construct it with a keyed literal. Do does not alter it.
type GenerateSentientItemTraitsCommand struct {
	RollIntelligence, RollWisdom, RollCharisma                bool
	RollAlignment, RollCommunication, RollSenses, RollPurpose bool

	Intelligence, Wisdom, Charisma ability.Score
	Alignment                      content.Alignment
	Communication                  content.SentientCommunication
	Senses                         content.SentientSenses
	Purpose                        SpecialPurposeSelection
	PurposeDetail                  PurposeDetail
}

// GenerateRejection is why generation was refused. The zero value is not a reason.
type GenerateRejection int

// The reasons.
const (
	GeneratePurposeDetailMissing GenerateRejection = iota + 1
	GenerateUnalignedNotAllowed
	GenerateAlignedPurposeInvalid
	GenerateAlignedPurposeNeedsNonNeutralAlignment
)

// String returns the snake_case name of the reason, or "unknown".
func (r GenerateRejection) String() string {
	names := []string{ValueRejectionPurposeDetailMissing, ValueRejectionUnalignedNotAllowed, ValueRejectionAlignedPurposeInvalid, ValueRejectionAlignedPurposeNeedsNonNeutralAlignment}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// GenerateSentientItemTraitsResult is the closed set of outcomes of
// GenerateSentientItemTraits.Do. Variants: GenerateSentientItemTraitsGenerated,
// GenerateSentientItemTraitsRejected.
type GenerateSentientItemTraitsResult interface{ isGenerateSentientItemTraitsResult() }

// GenerateSentientItemTraitsGenerated: the traits.
type GenerateSentientItemTraitsGenerated struct{ traits SentientTraits }

// Traits returns the generated traits.
func (r GenerateSentientItemTraitsGenerated) Traits() SentientTraits { return r.traits }

// GenerateSentientItemTraitsRejected: the traits cannot be used and the host
// re-chooses. Kind is the purpose kind and Alignment the alignment in play, for
// the reasons that need them. It is a legal roll, not an error.
type GenerateSentientItemTraitsRejected struct {
	reason    GenerateRejection
	kind      content.SpecialPurposeKind
	alignment content.Alignment
}

// Reason returns the reason.
func (r GenerateSentientItemTraitsRejected) Reason() GenerateRejection { return r.reason }

// PurposeKind returns the purpose kind in play, or zero.
func (r GenerateSentientItemTraitsRejected) PurposeKind() content.SpecialPurposeKind { return r.kind }

// Alignment returns the alignment in play, or zero.
func (r GenerateSentientItemTraitsRejected) Alignment() content.Alignment { return r.alignment }

func (GenerateSentientItemTraitsGenerated) isGenerateSentientItemTraitsResult() {}
func (GenerateSentientItemTraitsRejected) isGenerateSentientItemTraitsResult()  {}

// GenerateSentientItemTraits is the use case that rolls or takes the traits of a
// sentient item (SRD 5.2.1, "Sentient Magic Item Traits", pp. 207-208). Create it
// with NewGenerateSentientItemTraits; the zero value is not usable. It is safe for
// concurrent use only if its roller, telemetry and clock are.
type GenerateSentientItemTraits struct {
	roller Roller
	sig    operationSignals
}

// NewGenerateSentientItemTraits returns the use case rolling through roller.
// Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewGenerateSentientItemTraits(roller Roller, opts ...Option) (GenerateSentientItemTraits, error) {
	if roller == nil {
		return GenerateSentientItemTraits{}, invalidInput("generate sentient item traits without a dice roller")
	}
	return GenerateSentientItemTraits{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do makes one request for each trait the command asks to roll, in this order:
// Intelligence, Wisdom, Charisma (4d6, lowest die dropped), alignment (1d100),
// communication (1d10), senses (1d4), special purpose (1d10); none for a chosen
// trait. A chosen unaligned alignment is rejected before any request. A rolled
// purpose needing a detail the command did not supply is GeneratePurposeDetailMissing;
// an Aligned purpose with a Neutral item is
// GenerateAlignedPurposeNeedsNonNeutralAlignment (a legal roll the host
// re-chooses); an Aligned purpose that fails NewAlignedPurpose is
// GenerateAlignedPurposeInvalid.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled, and an error matching srderr.ErrInvalidInput when a chosen trait is
// unset. It returns a *srderr.PortError (error type "dice_roller_error") with a
// nil result when the roller fails; the failure is recorded once and never logged.
func (u GenerateSentientItemTraits) Do(ctx context.Context, cmd GenerateSentientItemTraitsCommand) (GenerateSentientItemTraitsResult, error) {
	if u.roller == nil {
		return nil, invalidInput("zero GenerateSentientItemTraits; use NewGenerateSentientItemTraits")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanGenerateSentientTraits, ValueOperationGenerateSentientTraits)
	defer run.end(ctx)
	reject := func(r GenerateRejection, kind content.SpecialPurposeKind, a content.Alignment) (GenerateSentientItemTraitsResult, error) {
		run.finish(ctx, ValueResultRejected, r.String())
		return GenerateSentientItemTraitsRejected{reason: r, kind: kind, alignment: a}, nil
	}
	if !cmd.RollAlignment && cmd.Alignment == content.AlignmentUnaligned {
		return reject(GenerateUnalignedNotAllowed, 0, cmd.Alignment)
	}
	roll := func(n int, die dice.Die) (dice.Result, error) {
		expr, err := dice.NewExpression(n, die, 0)
		if err != nil { // unreachable: fixed counts and standard dice
			return dice.Result{}, err
		}
		res, err := dice.Roll(u.roller, expr)
		if err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
		}
		return res, err
	}
	var (
		t   SentientTraits
		err error
		res dice.Result
	)
	scores := [3]ability.Score{cmd.Intelligence, cmd.Wisdom, cmd.Charisma}
	for i, rolled := range []bool{cmd.RollIntelligence, cmd.RollWisdom, cmd.RollCharisma} {
		if !rolled {
			continue
		}
		if res, err = roll(4, dice.D6()); err != nil {
			return nil, err
		}
		if scores[i], err = (SentientAbilityRoll{}).FromDice(res); err != nil { // unreachable: four dice
			return nil, err
		}
	}
	alignment, communication, senses, purpose := cmd.Alignment, cmd.Communication, cmd.Senses, cmd.Purpose
	if cmd.RollAlignment {
		if res, err = roll(1, dice.D100()); err != nil {
			return nil, err
		}
		if alignment, err = content.SentientAlignmentTableForRoll(res.Total()); err != nil { // unreachable: 1 to 100
			return nil, err
		}
	}
	if cmd.RollCommunication {
		if res, err = roll(1, dice.D10()); err != nil {
			return nil, err
		}
		if communication, err = content.SentientCommunicationForRoll(res.Total()); err != nil { // unreachable: 1 to 10
			return nil, err
		}
	}
	if cmd.RollSenses {
		if res, err = roll(1, dice.D4()); err != nil {
			return nil, err
		}
		if senses, err = content.SentientSensesForRoll(res.Total()); err != nil { // unreachable: 1 to 4
			return nil, err
		}
	}
	var kind content.SpecialPurposeKind
	if cmd.RollPurpose {
		if res, err = roll(1, dice.D10()); err != nil {
			return nil, err
		}
		if kind, err = content.SpecialPurposeKindForRoll(res.Total()); err != nil { // unreachable: 1 to 10
			return nil, err
		}
		var ok bool
		if purpose, ok = purposeOfKind(kind, alignment, cmd.PurposeDetail); !ok {
			return reject(GeneratePurposeDetailMissing, kind, alignment)
		}
	} else if p, ok := purpose.Purpose(); ok {
		kind = p.Kind()
	}
	if kind == content.SpecialPurposeAligned {
		if l, g, _ := axes(alignment); l == 0 && g == 0 {
			return reject(GenerateAlignedPurposeNeedsNonNeutralAlignment, kind, alignment)
		}
	}
	t, err = NewSentientTraits(content.MagicItemWondrousItem, scores[0], scores[1], scores[2], alignment, communication, senses, purpose)
	if err != nil {
		var rej *SentientTraitsRejected
		if errors.As(err, &rej) && rej.Reason == SentientAlignedPurposeInvalid {
			return reject(GenerateAlignedPurposeInvalid, kind, alignment)
		}
		return nil, err
	}
	run.finish(ctx, ValueResultGenerated, "")
	return GenerateSentientItemTraitsGenerated{traits: t}, nil
}
